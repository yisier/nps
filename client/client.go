package client

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"ehang.io/nps/lib/nps_mux"
	"github.com/pires/go-proxyproto"

	"github.com/astaxie/beego/logs"
	"github.com/xtaci/kcp-go"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/config"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/crypt"
)

type TRPClient struct {
	svrAddr        string
	bridgeConnType string
	proxyUrl       string
	vKey           string
	p2pAddr        map[string]string
	tunnel         *nps_mux.Mux
	signal         *conn.Conn
	ticker         *time.Ticker
	cnf            *config.Config
	disconnectTime int
	once           sync.Once
	logger         *logs.BeeLogger // 每个客户端独立的 logger
}

// new client
func NewRPClient(svraddr string, vKey string, bridgeConnType string, proxyUrl string, cnf *config.Config, disconnectTime int) *TRPClient {
	return &TRPClient{
		svrAddr:        svraddr,
		p2pAddr:        make(map[string]string, 0),
		vKey:           vKey,
		bridgeConnType: bridgeConnType,
		proxyUrl:       proxyUrl,
		cnf:            cnf,
		disconnectTime: disconnectTime,
		once:           sync.Once{},
		logger:         nil, // 默认使用全局 logger，可通过 SetLogger 设置
	}
}

// SetLogger 设置客户端的独立 logger
func (s *TRPClient) SetLogger(logger *logs.BeeLogger) {
	s.logger = logger
}

// log 辅助方法：如果设置了独立 logger 就使用，否则使用全局 logger
func (s *TRPClient) logInfo(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Info(format, v...)
	} else {
		logs.Info(format, v...)
	}
}

func (s *TRPClient) logError(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Error(format, v...)
	} else {
		logs.Error(format, v...)
	}
}

func (s *TRPClient) logWarn(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Warn(format, v...)
	} else {
		logs.Warn(format, v...)
	}
}

func (s *TRPClient) logTrace(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Trace(format, v...)
	} else {
		logs.Trace(format, v...)
	}
}

// IsConnected 返回客户端是否已成功连接到服务器
func (s *TRPClient) IsConnected() bool {
	return s.signal != nil
}

var NowStatus int
var CloseClient bool

// start
func (s *TRPClient) Start() {
	CloseClient = false
retry:
	if CloseClient {
		return
	}
	NowStatus = 0
	c, err := NewConn(s.bridgeConnType, s.vKey, s.svrAddr, common.WORK_MAIN, s.proxyUrl)
	if err != nil {
		s.logError("The connection server failed and will be reconnected in five seconds, error", err.Error())
		time.Sleep(time.Second * 5)
		goto retry
	}
	if c == nil {
		s.logError("Error data from server, and will be reconnected in five seconds")
		time.Sleep(time.Second * 5)
		goto retry
	}
	s.logInfo("Successful connection with server %s", s.svrAddr)
	//monitor the connection
	go s.ping()
	s.signal = c
	//start a channel connection
	go s.newChan()
	//start health check if the it's open
	if s.cnf != nil && len(s.cnf.Healths) > 0 {
		go heathCheck(s.cnf.Healths, s.signal)
	}
	NowStatus = 1
	//msg connection, eg udp
	s.handleMain()
}

// handle main connection
func (s *TRPClient) handleMain() {
	for {
		flags, err := s.signal.ReadFlag()
		if err != nil {
			s.logError("Accept server data error %s, end this service", err.Error())
			break
		}
		switch flags {
		case common.NEW_UDP_CONN:
			//read server udp addr and password
			if lAddr, err := s.signal.GetShortLenContent(); err != nil {
				s.logWarn(err.Error())
				return
			} else if pwd, err := s.signal.GetShortLenContent(); err == nil {
				var localAddr string
				//The local port remains unchanged for a certain period of time
				if v, ok := s.p2pAddr[crypt.Md5(string(pwd)+strconv.Itoa(int(time.Now().Unix()/100)))]; !ok {
					tmpConn, err := common.GetLocalUdpAddr()
					if err != nil {
						s.logError(err.Error())
						return
					}
					localAddr = tmpConn.LocalAddr().String()
				} else {
					localAddr = v
				}
				go s.newUdpConn(localAddr, string(lAddr), string(pwd))
			}
		}
	}
	s.Close()
}

func (s *TRPClient) newUdpConn(localAddr, rAddr string, md5Password string) {
	var localConn net.PacketConn
	var err error
	var remoteAddress string
	if remoteAddress, localConn, err = handleP2PUdp(localAddr, rAddr, md5Password, common.WORK_P2P_PROVIDER); err != nil {
		s.logError(err.Error())
		return
	}
	l, err := kcp.ServeConn(nil, 150, 3, localConn)
	if err != nil {
		s.logError(err.Error())
		return
	}
	s.logTrace("start local p2p udp listen, local address %s", localConn.LocalAddr().String())
	for {
		udpTunnel, err := l.AcceptKCP()
		if err != nil {
			s.logError(err.Error())
			l.Close()
			return
		}
		if udpTunnel.RemoteAddr().String() == string(remoteAddress) {
			conn.SetUdpSession(udpTunnel)
			s.logTrace("successful connection with client ,address %s", udpTunnel.RemoteAddr().String())
			//read link info from remote
			conn.Accept(nps_mux.NewMux(udpTunnel, s.bridgeConnType, s.disconnectTime), func(c net.Conn) {
				go s.handleChan(c)
			})
			break
		}
	}
}

// pmux tunnel
func (s *TRPClient) newChan() {
	tunnel, err := NewConn(s.bridgeConnType, s.vKey, s.svrAddr, common.WORK_CHAN, s.proxyUrl)
	if err != nil {
		s.logError("connect to %s error: %v", s.svrAddr, err)
		return
	}
	s.tunnel = nps_mux.NewMux(tunnel.Conn, s.bridgeConnType, s.disconnectTime)
	for {
		src, err := s.tunnel.Accept()
		if err != nil {
			s.logWarn(err.Error())
			s.Close()
			break
		}
		go s.handleChan(src)
	}
}

func (s *TRPClient) handleChan(src net.Conn) {
	lk, err := conn.NewConn(src).GetLinkInfo()
	if err != nil || lk == nil {
		src.Close()
		s.logError("get connection info from server error %v", err)
		return
	}
	//host for target processing
	lk.Host = common.FormatAddress(lk.Host)
	//if Conn type is http, read the request and log
	if lk.ConnType == "http" {
		if targetConn, err := net.DialTimeout(common.CONN_TCP, lk.Host, lk.Option.Timeout); err != nil {
			s.logWarn("connect to %s error %s", lk.Host, err.Error())
			src.Close()
		} else {
			srcConn := conn.GetConn(src, lk.Crypt, lk.Compress, nil, false)
			go func() {
				common.CopyBuffer(srcConn, targetConn)
				srcConn.Close()
				targetConn.Close()
			}()
			for {
				if r, err := http.ReadRequest(bufio.NewReader(srcConn)); err != nil {
					srcConn.Close()
					targetConn.Close()
					break
				} else {
					remoteAddr := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
					if len(remoteAddr) == 0 {
						remoteAddr = r.RemoteAddr
					}
					s.logTrace("http request, method %s, host %s, url %s, remote address %s", r.Method, r.Host, r.URL.Path, remoteAddr)
					r.Write(targetConn)
				}
			}
		}
		return
	}
	if lk.ConnType == "udp5" {
		s.logTrace("new %s connection with the goal of %s, remote address:%s", lk.ConnType, lk.Host, lk.RemoteAddr)
		s.handleUdp(src)
	}
	//connect to target if conn type is tcp or udp
	if targetConn, err := net.DialTimeout(lk.ConnType, lk.Host, lk.Option.Timeout); err != nil {
		s.logWarn("connect to %s error %s", lk.Host, err.Error())
		src.Close()
	} else {
		s.logTrace("new %s connection with the goal of %s, remote address:%s", lk.ConnType, lk.Host, lk.RemoteAddr)

		if lk.ProtoVersion == "V1" || lk.ProtoVersion == "V2" {
			var addr = targetConn.RemoteAddr()
			if lk.RemoteAddr != "" {
				s := strings.Split(lk.RemoteAddr, ":")[1]
				port, _ := strconv.Atoi(s)
				addr = &net.TCPAddr{
					IP:   net.ParseIP(strings.Split(lk.RemoteAddr, ":")[0]),
					Port: port,
				}
			}

			var version byte

			if lk.ProtoVersion == "V1" {
				version = 1
			} else if lk.ProtoVersion == "V2" {
				version = 2
			}

			transportProtocol := proxyproto.TCPv4
			if strings.Contains(addr.String(), ".") {
				transportProtocol = proxyproto.TCPv4
			} else {
				transportProtocol = proxyproto.TCPv6
			}

			header := &proxyproto.Header{
				Command:           proxyproto.PROXY,
				SourceAddr:        addr,
				DestinationAddr:   targetConn.RemoteAddr(),
				Version:           version,
				TransportProtocol: transportProtocol,
			}

			_, err2 := header.WriteTo(targetConn)
			if err2 != nil {
				s.logError(err2.Error())
			}
		}

		conn.CopyWaitGroup(src, targetConn, lk.Crypt, lk.Compress, nil, nil, false, nil, nil)
	}
}

func (s *TRPClient) handleUdp(serverConn net.Conn) {
	// bind a local udp port
	local, err := net.ListenUDP("udp", nil)
	defer serverConn.Close()
	if err != nil {
		s.logError("bind local udp port error %s", err.Error())
		return
	}
	defer local.Close()
	go func() {
		defer serverConn.Close()
		b := common.BufPoolUdp.Get().([]byte)
		defer common.BufPoolUdp.Put(b)
		for {
			n, raddr, err := local.ReadFrom(b)
			if err != nil {
				s.logError("read data from remote server error %s", err.Error())
			}
			buf := bytes.Buffer{}
			dgram := common.NewUDPDatagram(common.NewUDPHeader(0, 0, common.ToSocksAddr(raddr)), b[:n])
			dgram.Write(&buf)
			b, err := conn.GetLenBytes(buf.Bytes())
			if err != nil {
				s.logWarn("get len bytes error %s", err.Error())
				continue
			}
			if _, err := serverConn.Write(b); err != nil {
				s.logError("write data to remote  error %s", err.Error())
				return
			}
		}
	}()
	b := common.BufPoolUdp.Get().([]byte)
	defer common.BufPoolUdp.Put(b)
	for {
		n, err := serverConn.Read(b)
		if err != nil {
			s.logError("read udp data from server error %s", err.Error())
			return
		}

		udpData, err := common.ReadUDPDatagram(bytes.NewReader(b[:n]))
		if err != nil {
			s.logError("unpack data error %s", err.Error())
			return
		}
		raddr, err := net.ResolveUDPAddr("udp", udpData.Header.Addr.String())
		if err != nil {
			s.logError("build remote addr err %s", err.Error())
			continue // drop silently
		}
		_, err = local.WriteTo(udpData.Data, raddr)
		if err != nil {
			s.logError("write data to remote %s error %s", raddr.String(), err.Error())
			return
		}
	}
}

// Whether the monitor channel is closed
func (s *TRPClient) ping() {
	s.ticker = time.NewTicker(time.Second * 5)
loop:
	for {
		select {
		case <-s.ticker.C:
			if s.tunnel != nil && s.tunnel.IsClose {
				s.Close()
				break loop
			}
		}
	}
}

func (s *TRPClient) Close() {
	s.once.Do(s.closing)
}

func (s *TRPClient) closing() {
	CloseClient = true
	NowStatus = 0
	if s.tunnel != nil {
		_ = s.tunnel.Close()
	}
	if s.signal != nil {
		_ = s.signal.Close()
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
}
