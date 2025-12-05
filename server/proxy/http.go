package proxy

import (
	"bufio"
	"crypto/tls"
	"ehang.io/nps/bridge"
	"ehang.io/nps/lib/cache"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/lib/goroutine"
	"ehang.io/nps/server/connection"
	"encoding/json"
	"github.com/astaxie/beego/logs"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type httpServer struct {
	BaseServer
	httpPort      int
	httpsPort     int
	httpServer    *http.Server
	httpsServer   *http.Server
	httpsListener net.Listener
	useCache      bool
	addOrigin     bool
	cache         *cache.Cache
	cacheLen      int
}

func NewHttp(bridge *bridge.Bridge, c *file.Tunnel, httpPort, httpsPort int, useCache bool, cacheLen int, addOrigin bool) *httpServer {
	httpServer := &httpServer{
		BaseServer: BaseServer{
			task:   c,
			bridge: bridge,
			Mutex:  sync.Mutex{},
		},
		httpPort:  httpPort,
		httpsPort: httpsPort,
		useCache:  useCache,
		cacheLen:  cacheLen,
		addOrigin: addOrigin,
	}
	if useCache {
		httpServer.cache = cache.New(cacheLen)
	}
	return httpServer
}

func (s *httpServer) Start() error {
	var err error
	if s.errorContent, err = common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "web", "static", "page", "error.html")); err != nil {
		s.errorContent = []byte("nps 404")
	}
	if s.httpPort > 0 {
		s.httpServer = s.NewServer(s.httpPort, "http")
		go func() {
			l, err := connection.GetHttpListener()
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
			err = s.httpServer.Serve(l)
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
		}()
	}
	if s.httpsPort > 0 {
		s.httpsServer = s.NewServer(s.httpsPort, "https")
		go func() {
			s.httpsListener, err = connection.GetHttpsListener()
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
			logs.Error(NewHttpsServer(s.httpsListener, s.bridge, s.useCache, s.cacheLen).Start())
		}()
	}
	return nil
}

func (s *httpServer) Close() error {
	if s.httpsListener != nil {
		s.httpsListener.Close()
	}
	if s.httpsServer != nil {
		s.httpsServer.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	return nil
}

func (s *httpServer) handleTunneling(w http.ResponseWriter, r *http.Request) {

	var host *file.Host
	var err error
	host, err = file.GetDb().GetInfoByHost(r.Host, r)
	if err != nil {
		logs.Debug("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
		return
	}

	// 自动 http 301 https
	if host.AutoHttps && r.TLS == nil {
		http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
		return
	}

	if r.Header.Get("Upgrade") != "" {
		rProxy := NewHttpReverseProxy(s)
		rProxy.ServeHTTP(w, r)
	} else {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}
		c, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}

		s.handleHttp(conn.NewConn(c), r)
	}

}

func (s *httpServer) handleHttp(c *conn.Conn, r *http.Request) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		connClient io.ReadWriteCloser
		scheme     = r.URL.Scheme
		lk         *conn.Link
		targetAddr string
		lenConn    *conn.LenConn
		isReset    bool
		wg         sync.WaitGroup
		remoteAddr string
	)
	defer func() {
		if connClient != nil {
			connClient.Close()
		} else {
			s.writeConnFail(c.Conn)
		}
		c.Close()
	}()
reset:
	if isReset {
		host.Client.AddConn()
	}

	remoteAddr = strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if len(remoteAddr) == 0 {
		remoteAddr = c.RemoteAddr().String()
	}

	// 判断访问地址是否在全局黑名单内
	if IsGlobalBlackIp(c.RemoteAddr().String()) {
		c.Close()
		return
	}

	if host, err = file.GetDb().GetInfoByHost(r.Host, r); err != nil {
		logs.Notice("the url %s %s %s can't be parsed!, host %s, url %s, remote address %s", r.URL.Scheme, r.Host, r.RequestURI, r.Host, r.URL.Path, remoteAddr)
		c.Close()
		return
	}

	if err := s.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		c.Close()
		return
	}
	if !isReset {
		defer host.Client.AddConn()
	}
	if err = s.auth(r, c, host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
		logs.Warn("auth error", err, r.RemoteAddr)
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		logs.Warn(err.Error())
		return
	}

	if host.Client.IpWhite && host.Client.IpWhitePass != "" {
		if common.IsAuthIp(c.RemoteAddr().String(), host.Client.VerifyKey, host.Client.IpWhiteList) {

			if r.URL.Path == "/authIp" {
				rawQuery := r.URL.RawQuery

				var jsonBytes []byte
				ip := common.GetIpByAddr(c.RemoteAddr().String())

				if strings.Contains(rawQuery, "pass=") {
					// 截取 = 后面的值
					pass := strings.Split(rawQuery, "pass=")[1]
					if pass == host.Client.IpWhitePass {
						host.Client.IpWhiteList = append(host.Client.IpWhiteList, ip)
						file.GetDb().UpdateClient(host.Client)
						logs.Info("客户端IP白名单认证授权成功:vkey [%s] ip [%s] password [%s]", host.Client.VerifyKey, ip, pass)
						jsonBytes, err = json.Marshal(map[string]interface{}{"success": true, "message": "授权成功"})
					} else {
						logs.Error("客户端IP白名单认证授权密码错误:vkey [%s] ip [%s] password [%s]", host.Client.VerifyKey, ip, pass)
						jsonBytes, err = json.Marshal(map[string]interface{}{"success": false, "message": "密码错误"})
					}
				} else {
					logs.Error("客户端IP白名单认证授权密码错误:vkey [%s] ip [%s]", host.Client.VerifyKey, ip)
					jsonBytes, err = json.Marshal(map[string]interface{}{"success": false, "message": "参数错误"})
				}
				s.errorContent, err = jsonBytes, err
				s.errorCode = 200
				return
			}

			errorContent, _ := common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "web", "static", "page", "auth.html"))
			authHtml := string(errorContent)
			authHtml = strings.ReplaceAll(authHtml, "${ip}", common.GetIpByAddr(c.RemoteAddr().String()))
			s.errorContent, err = []byte(authHtml), err
			s.errorCode = 401
			return
		}
	}

	// 判断访问地址是否在黑名单内
	if common.IsBlackIp(c.RemoteAddr().String(), host.Client.VerifyKey, host.Client.BlackIpList) {
		c.Close()
		return
	}

	lk = conn.NewLink("http", targetAddr, host.Client.Cnf.Crypt, host.Client.Cnf.Compress, r.RemoteAddr, host.Target.LocalProxy, "")
	if target, err = s.bridge.SendLinkInfo(host.Client.Id, lk, nil); err != nil {
		logs.Notice("connect to target %s error %s", lk.Host, err)
		return
	}
	connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)

	//read from inc-client
	go func() {
		wg.Add(1)
		isReset = false
		defer connClient.Close()
		defer func() {
			wg.Done()
			if !isReset {
				c.Close()
			}
		}()

		err1 := goroutine.CopyBuffer(c, connClient, host.Client.Flow, nil, "")
		if err1 != nil {
			return
		}

		resp, err := http.ReadResponse(bufio.NewReader(connClient), r)
		if err != nil || resp == nil || r == nil {
			// if there got broken pipe, http.ReadResponse will get a nil
			//break
			return
		} else {
			lenConn := conn.NewLenConn(c)
			if err := resp.Write(lenConn); err != nil {
				logs.Error(err)
				//break
				return
			}
		}
	}()

	for {
		//if the cache start and the request is in the cache list, return the cache
		if s.useCache {
			if v, ok := s.cache.Get(filepath.Join(host.Host, r.URL.Path)); ok {
				n, err := c.Write(v.([]byte))
				if err != nil {
					break
				}
				logs.Trace("%s request, method %s, host %s, url %s, remote address %s, return cache", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String())
				host.Client.Flow.Add(int64(n), int64(n))
				//if return cache and does not create a new conn with client and Connection is not set or close, close the connection.
				if strings.ToLower(r.Header.Get("Connection")) == "close" || strings.ToLower(r.Header.Get("Connection")) == "" {
					break
				}
				goto readReq
			}
		}

		//change the host and header and set proxy setting
		common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, c.Conn.RemoteAddr().String())

		logs.Info("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, remoteAddr, lk.Host)

		//write
		lenConn = conn.NewLenConn(connClient)
		//lenConn = conn.LenConn
		if err := r.Write(lenConn); err != nil {
			logs.Error(err)
			break
		}
		host.Client.Flow.Add(int64(lenConn.Len), int64(lenConn.Len))

	readReq:
		//read req from connection
		r, err = http.ReadRequest(bufio.NewReader(c))
		if err != nil {
			//break
			return
		}
		r.URL.Scheme = scheme
		//What happened ，Why one character less???
		r.Method = resetReqMethod(r.Method)
		if hostTmp, err := file.GetDb().GetInfoByHost(r.Host, r); err != nil {
			logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
			break
		} else if host != hostTmp {
			host = hostTmp
			isReset = true
			connClient.Close()
			goto reset
		}
	}
	wg.Wait()
}

func resetReqMethod(method string) string {
	if method == "ET" {
		return "GET"
	}
	if method == "OST" {
		return "POST"
	}
	return method
}

func (s *httpServer) NewServer(port int, scheme string) *http.Server {
	return &http.Server{
		Addr: ":" + strconv.Itoa(port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = scheme
			s.handleTunneling(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
}

func (s *httpServer) NewServerWithTls(port int, scheme string, l net.Listener, certFile string, keyFile string) error {

	if certFile == "" || keyFile == "" {
		logs.Error("证书文件为空")
		return nil
	}
	var certFileByte = []byte(certFile)
	var keyFileByte = []byte(keyFile)

	config := &tls.Config{}
	config.Certificates = make([]tls.Certificate, 1)

	var err error
	config.Certificates[0], err = tls.X509KeyPair(certFileByte, keyFileByte)
	if err != nil {
		return err
	}

	s2 := &http.Server{
		Addr: ":" + strconv.Itoa(port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = scheme
			s.handleTunneling(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		TLSConfig:    config,
	}

	return s2.ServeTLS(l, "", "")
}
