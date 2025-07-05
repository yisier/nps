package proxy

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"ehang.io/nps/lib/cache"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/crypt"
	"ehang.io/nps/lib/file"
	"github.com/astaxie/beego/logs"
	"github.com/pkg/errors"
)

type HttpsServer struct {
	httpServer
	listener         net.Listener
	httpsListenerMap sync.Map
	hostIdCertMap    sync.Map
}

func NewHttpsServer(l net.Listener, bridge NetBridge, useCache bool, cacheLen int) *HttpsServer {
	https := &HttpsServer{listener: l}
	https.bridge = bridge
	https.useCache = useCache
	if useCache {
		https.cache = cache.New(cacheLen)
	}
	return https
}

// start https server
func (https *HttpsServer) Start() error {

	conn.Accept(https.listener, func(c net.Conn) {
		serverName, rb := GetServerNameFromClientHello(c)
		r := buildHttpsRequest(serverName)
		if host, err := file.GetDb().GetInfoByHost(serverName, r); err != nil {
			c.Close()
			logs.Debug("the url %s can't be parsed!,remote addr %s", serverName, c.RemoteAddr().String())
			return
		} else {
			if host.CertFilePath == "" || host.KeyFilePath == "" {
				logs.Debug("加载客户端本地证书")
				https.handleHttps2(c, serverName, rb, r)
			} else {
				logs.Debug("使用上传证书")

				// 判断是路径还是证书，-----BEGIN 开头的为证书
				if strings.Contains(host.CertFilePath, "-----BEGIN") || strings.Contains(host.KeyFilePath, "-----BEGIN") {
					logs.Debug("通过上传文件加载证书")
					https.cert(host, c, rb, host.CertFilePath, host.KeyFilePath)
				} else {
					logs.Debug("通过路径加载证书")
					if !common.FileExists(host.CertFilePath) || !common.FileExists(host.KeyFilePath) {
						c.Close()
						logs.Error("证书或秘钥文件不存在", host.KeyFilePath, host.CertFilePath)
						return
					}

					cert, err := common.ReadAllFromFile(host.CertFilePath)
					if err != nil {
						c.Close()
						logs.Error("加载证书失败", err)
						return
					}
					key, err := common.ReadAllFromFile(host.KeyFilePath)
					if err != nil {
						c.Close()
						logs.Error("加载证书秘钥失败", err)
						return
					}

					https.cert(host, c, rb, string(cert), string(key))

				}

			}
		}
	})

	//var err error
	//if https.errorContent, err = common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "web", "static", "page", "error.html")); err != nil {
	//	https.errorContent = []byte("nps 404")
	//}
	//if b, err := beego.AppConfig.Bool("https_just_proxy"); err == nil && b {
	//	conn.Accept(https.listener, func(c net.Conn) {
	//		https.handleHttps(c)
	//	})
	//} else {
	//	//start the default listener
	//	certFile := beego.AppConfig.String("https_default_cert_file")
	//	keyFile := beego.AppConfig.String("https_default_key_file")
	//	if common.FileExists(certFile) && common.FileExists(keyFile) {
	//		l := NewHttpsListener(https.listener)
	//		https.NewHttps(l, certFile, keyFile)
	//		https.httpsListenerMap.Store("default", l)
	//	}
	//	conn.Accept(https.listener, func(c net.Conn) {
	//		serverName, rb := GetServerNameFromClientHello(c)
	//		//if the clientHello does not contains sni ,use the default ssl certificate
	//		if serverName == "" {
	//			serverName = "default"
	//		}
	//		var l *HttpsListener
	//		if v, ok := https.httpsListenerMap.Load(serverName); ok {
	//			l = v.(*HttpsListener)
	//		} else {
	//			r := buildHttpsRequest(serverName)
	//			if host, err := file.GetDb().GetInfoByHost(serverName, r); err != nil {
	//				c.Close()
	//				logs.Notice("the url %s can't be parsed!,remote addr %s", serverName, c.RemoteAddr().String())
	//				return
	//			} else {
	//				if !common.FileExists(host.CertFilePath) || !common.FileExists(host.KeyFilePath) {
	//					//if the host cert file or key file is not set ,use the default file
	//					if v, ok := https.httpsListenerMap.Load("default"); ok {
	//						l = v.(*HttpsListener)
	//					} else {
	//						c.Close()
	//						logs.Error("the key %s cert %s file is not exist", host.KeyFilePath, host.CertFilePath)
	//						return
	//					}
	//				} else {
	//					l = NewHttpsListener(https.listener)
	//					https.NewHttps(l, host.CertFilePath, host.KeyFilePath)
	//					https.httpsListenerMap.Store(serverName, l)
	//				}
	//			}
	//		}
	//		acceptConn := conn.NewConn(c)
	//		acceptConn.Rb = rb
	//		l.acceptConn <- acceptConn
	//	})
	//}
	return nil
}

func (https *HttpsServer) cert(host *file.Host, c net.Conn, rb []byte, certFileUrl string, keyFileUrl string) {
	var l *HttpsListener
	i := 0
	https.hostIdCertMap.Range(func(key, value interface{}) bool {
		i++
		// 如果host Id 不存在，则删除map
		if id, ok := key.(int); ok {
			var err error
			_, err = file.GetDb().GetHostById(id)
			if err != nil {
				// 说明这个host已经不存了，需要释放Listener
				logs.Error(err)
				if oldL, ok := https.httpsListenerMap.Load(value); ok {
					err := oldL.(*HttpsListener).Close()
					if err != nil {
						logs.Error(err)
					}
					https.httpsListenerMap.Delete(value)
					https.hostIdCertMap.Delete(key)
					logs.Info("Listener 已释放")
				}
			}
		}
		return true
	})

	logs.Info("当前 Listener 连接数量", i)

	if cert, ok := https.hostIdCertMap.Load(host.Id); ok {
		if cert == certFileUrl {
			// 证书已经存在，直接加载
			if v, ok := https.httpsListenerMap.Load(certFileUrl); ok {
				l = v.(*HttpsListener)
			}
		} else {
			// 证书修改过，重新加载证书
			l = NewHttpsListener(https.listener)
			https.NewHttps(l, certFileUrl, keyFileUrl)
			if oldL, ok := https.httpsListenerMap.Load(cert); ok {
				err := oldL.(*HttpsListener).Close()
				if err != nil {
					logs.Error(err)
				}
				https.httpsListenerMap.Delete(cert)
			}
			https.httpsListenerMap.Store(certFileUrl, l)
			https.hostIdCertMap.Store(host.Id, certFileUrl)
		}
	} else {
		// 第一次加载证书
		l = NewHttpsListener(https.listener)
		https.NewHttps(l, certFileUrl, keyFileUrl)
		https.httpsListenerMap.Store(certFileUrl, l)
		https.hostIdCertMap.Store(host.Id, certFileUrl)
	}

	acceptConn := conn.NewConn(c)
	acceptConn.Rb = rb
	l.acceptConn <- acceptConn
}

// handle the https which is just proxy to other client
func (https *HttpsServer) handleHttps2(c net.Conn, hostName string, rb []byte, r *http.Request) {
	var targetAddr string
	var host *file.Host
	var err error
	if host, err = file.GetDb().GetInfoByHost(hostName, r); err != nil {
		c.Close()
		logs.Debug("the url %s can't be parsed!", hostName)
		return
	}
	if err := https.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Debug("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		c.Close()
		return
	}
	defer host.Client.AddConn()
	if err = https.auth(r, conn.NewConn(c), host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
		logs.Warn("auth error", err, r.RemoteAddr)
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		logs.Warn(err.Error())
	}
	logs.Info("new https connection,clientId %d,host %s,remote address %s", host.Client.Id, r.Host, c.RemoteAddr().String())
	https.DealClient(conn.NewConn(c), host.Client, targetAddr, rb, common.CONN_TCP, nil, host.Client.Flow, host.Target.LocalProxy, nil)
}

// close
func (https *HttpsServer) Close() error {
	return https.listener.Close()
}

// new https server by cert and key file
func (https *HttpsServer) NewHttps(l net.Listener, certFile string, keyFile string) {
	go func() {
		//logs.Error(https.NewServer(0, "https").ServeTLS(l, certFile, keyFile))
		logs.Error(https.NewServerWithTls(0, "https", l, certFile, keyFile))

	}()
}

// handle the https which is just proxy to other client
func (https *HttpsServer) handleHttps(c net.Conn) {
	hostName, rb := GetServerNameFromClientHello(c)
	var targetAddr string
	r := buildHttpsRequest(hostName)
	var host *file.Host
	var err error
	if host, err = file.GetDb().GetInfoByHost(hostName, r); err != nil {
		c.Close()
		logs.Notice("the url %s can't be parsed!", hostName)
		return
	}
	if err := https.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		c.Close()
		return
	}
	defer host.Client.AddConn()
	if err = https.auth(r, conn.NewConn(c), host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
		logs.Warn("auth error", err, r.RemoteAddr)
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		logs.Warn(err.Error())
	}
	logs.Trace("new https connection,clientId %d,host %s,remote address %s", host.Client.Id, r.Host, c.RemoteAddr().String())
	https.DealClient(conn.NewConn(c), host.Client, targetAddr, rb, common.CONN_TCP, nil, host.Client.Flow, host.Target.LocalProxy, nil)
}

type HttpsListener struct {
	acceptConn     chan *conn.Conn
	parentListener net.Listener
}

// https listener
func NewHttpsListener(l net.Listener) *HttpsListener {
	return &HttpsListener{parentListener: l, acceptConn: make(chan *conn.Conn)}
}

// accept
func (httpsListener *HttpsListener) Accept() (net.Conn, error) {
	httpsConn := <-httpsListener.acceptConn
	if httpsConn == nil {
		return nil, errors.New("get connection error")
	}
	return httpsConn, nil
}

// close
func (httpsListener *HttpsListener) Close() error {
	return nil
}

// addr
func (httpsListener *HttpsListener) Addr() net.Addr {
	return httpsListener.parentListener.Addr()
}

// get server name from connection by read client hello bytes
func GetServerNameFromClientHello(c net.Conn) (string, []byte) {
	buf := make([]byte, 4096)
	data := make([]byte, 4096)
	n, err := c.Read(buf)
	if err != nil {
		return "", nil
	}
	if n < 42 {
		return "", nil
	}
	copy(data, buf[:n])
	clientHello := new(crypt.ClientHelloMsg)
	clientHello.Unmarshal(data[5:n])
	return clientHello.GetServerName(), buf[:n]
}

// build https request
func buildHttpsRequest(hostName string) *http.Request {
	r := new(http.Request)
	r.RequestURI = "/"
	r.URL = new(url.URL)
	r.URL.Scheme = "https"
	r.Host = hostName
	return r
}
