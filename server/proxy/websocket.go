package proxy

import (
	"context"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/lib/goroutine"
	"errors"
	"github.com/astaxie/beego/logs"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
)

type HTTPError struct {
	error
	HTTPCode int
}

type HttpReverseProxy struct {
	proxy                 *ReverseProxy
	responseHeaderTimeout time.Duration
}
type flowConn struct {
	io.ReadWriteCloser
	fakeAddr net.Addr
	host     *file.Host
	flowIn   int64
	flowOut  int64
	once     sync.Once
}

func (rp *HttpReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var (
		host       *file.Host
		targetAddr string
		err        error
	)
	if host, err = file.GetDb().GetInfoByHost(req.Host, req); err != nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte(req.Host + " not found"))
		return
	}
	if host.Client.Cnf.U != "" && host.Client.Cnf.P != "" && !common.CheckAuth(req, host.Client.Cnf.U, host.Client.Cnf.P) {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Unauthorized"))
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte("502 Bad Gateway"))
		return
	}
	host.Client.CutConn()

	req = req.WithContext(context.WithValue(req.Context(), "host", host))
	req = req.WithContext(context.WithValue(req.Context(), "target", targetAddr))
	req = req.WithContext(context.WithValue(req.Context(), "req", req))

	rp.proxy.ServeHTTP(rw, req, host)

	defer host.Client.AddConn()
}

func (c *flowConn) Read(p []byte) (n int, err error) {
	n, err = c.ReadWriteCloser.Read(p)
	return n, err
}

func (c *flowConn) Write(p []byte) (n int, err error) {
	n, err = c.ReadWriteCloser.Write(p)
	return n, err
}

func (c *flowConn) Close() error {
	//c.once.Do(func() { c.host.Flow.Add(c.flowIn, c.flowOut) })
	return c.ReadWriteCloser.Close()
}

func (c *flowConn) LocalAddr() net.Addr { return c.fakeAddr }

func (c *flowConn) RemoteAddr() net.Addr { return c.fakeAddr }

func (*flowConn) SetDeadline(t time.Time) error { return nil }

func (*flowConn) SetReadDeadline(t time.Time) error { return nil }

func (*flowConn) SetWriteDeadline(t time.Time) error { return nil }

func NewHttpReverseProxy(s *httpServer) *HttpReverseProxy {
	rp := &HttpReverseProxy{
		responseHeaderTimeout: 30 * time.Second,
	}
	local, _ := net.ResolveTCPAddr("tcp", "127.0.0.1")
	proxy := NewReverseProxy(&httputil.ReverseProxy{
		Director: func(r *http.Request) {
			host := r.Context().Value("host").(*file.Host)
			common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, "")
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: rp.responseHeaderTimeout,
			DisableKeepAlives:     true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var (
					host       *file.Host
					target     net.Conn
					err        error
					connClient io.ReadWriteCloser
					targetAddr string
					lk         *conn.Link
				)

				r := ctx.Value("req").(*http.Request)
				host = ctx.Value("host").(*file.Host)
				targetAddr = ctx.Value("target").(string)

				lk = conn.NewLink("http", targetAddr, host.Client.Cnf.Crypt, host.Client.Cnf.Compress, r.RemoteAddr, host.Target.LocalProxy, "")
				if target, err = s.bridge.SendLinkInfo(host.Client.Id, lk, nil); err != nil {
					logs.Notice("connect to target %s error %s", lk.Host, err)
					return nil, NewHTTPError(http.StatusBadGateway, "Cannot connect to the server")
				}
				connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
				return &flowConn{
					ReadWriteCloser: connClient,
					fakeAddr:        local,
					host:            host,
				}, nil
			},
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			logs.Warn("do http proxy request error: %v", err)
			rw.WriteHeader(http.StatusNotFound)
		},
	})
	proxy.WebSocketDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var (
			host       *file.Host
			target     net.Conn
			err        error
			connClient io.ReadWriteCloser
			targetAddr string
			lk         *conn.Link
		)
		r := ctx.Value("req").(*http.Request)
		host = ctx.Value("host").(*file.Host)
		targetAddr = ctx.Value("target").(string)

		lk = conn.NewLink("tcp", targetAddr, host.Client.Cnf.Crypt, host.Client.Cnf.Compress, r.RemoteAddr, host.Target.LocalProxy, "")
		if target, err = s.bridge.SendLinkInfo(host.Client.Id, lk, nil); err != nil {
			logs.Notice("connect to target %s error %s", lk.Host, err)
			return nil, NewHTTPError(http.StatusBadGateway, "Cannot connect to the target")
		}
		connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
		return &flowConn{
			ReadWriteCloser: connClient,
			fakeAddr:        local,
			host:            host,
		}, nil
	}
	rp.proxy = proxy
	return rp
}

func NewHTTPError(code int, errmsg string) error {
	return &HTTPError{
		error:    errors.New(errmsg),
		HTTPCode: code,
	}
}

type ReverseProxy struct {
	*httputil.ReverseProxy
	WebSocketDialContext func(ctx context.Context, network, addr string) (net.Conn, error)
}

func IsWebsocketRequest(req *http.Request) bool {
	containsHeader := func(name, value string) bool {
		items := strings.Split(req.Header.Get(name), ",")
		for _, item := range items {
			if value == strings.ToLower(strings.TrimSpace(item)) {
				return true
			}
		}
		return false
	}
	return containsHeader("Connection", "upgrade") && containsHeader("Upgrade", "websocket")
}

func NewReverseProxy(orp *httputil.ReverseProxy) *ReverseProxy {
	rp := &ReverseProxy{
		ReverseProxy:         orp,
		WebSocketDialContext: nil,
	}
	rp.ErrorHandler = rp.errHandler
	return rp
}

func (p *ReverseProxy) errHandler(rw http.ResponseWriter, r *http.Request, e error) {
	if e == io.EOF {
		rw.WriteHeader(521)
		//rw.Write(getWaitingPageContent())
	} else {
		if httperr, ok := e.(*HTTPError); ok {
			rw.WriteHeader(httperr.HTTPCode)
		} else {
			rw.WriteHeader(http.StatusNotFound)
		}
		rw.Write([]byte("error: " + e.Error()))
	}
}

func (p *ReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request, host *file.Host) {
	if IsWebsocketRequest(req) {
		p.serveWebSocket(rw, req, host)
	}
}

func (p *ReverseProxy) serveWebSocket(rw http.ResponseWriter, req *http.Request, host *file.Host) {
	if p.WebSocketDialContext == nil {
		rw.WriteHeader(500)
		return
	}
	targetConn, err := p.WebSocketDialContext(req.Context(), "tcp", "")
	if err != nil {
		rw.WriteHeader(501)
		return
	}
	defer targetConn.Close()

	p.Director(req)

	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(500)
		return
	}
	conn, _, errHijack := hijacker.Hijack()
	if errHijack != nil {
		rw.WriteHeader(500)
		return
	}
	defer conn.Close()

	req.Write(targetConn)

	Join(conn, targetConn, host)
}

func Join(c1 io.ReadWriteCloser, c2 io.ReadWriteCloser, host *file.Host) (inCount int64, outCount int64) {
	var wait sync.WaitGroup
	pipe := func(to io.ReadWriteCloser, from io.ReadWriteCloser, count *int64) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()
		goroutine.CopyBuffer(to, from, host.Client.Flow, nil, "")
		//*count, _ = io.Copy(to, from)
	}

	wait.Add(2)

	go pipe(c1, c2, &inCount)
	go pipe(c2, c1, &outCount)
	wait.Wait()
	return
}
