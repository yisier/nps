package proxy

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"github.com/astaxie/beego/logs"
)

const (
	ipV4            = 1
	domainName      = 3
	ipV6            = 4
	connectMethod   = 1
	bindMethod      = 2
	associateMethod = 3
	// The maximum packet size of any udp Associate packet, based on ethernet's max size,
	// minus the IP and UDP headers. IPv4 has a 20 byte header, UDP adds an
	// additional 4 bytes.  This is a total overhead of 24 bytes.  Ethernet's
	// max packet size is 1500 bytes,  1500 - 24 = 1476.
	maxUDPPacketSize = 1476
)

const (
	succeeded uint8 = iota
	serverFailure
	notAllowed
	networkUnreachable
	hostUnreachable
	connectionRefused
	ttlExpired
	commandNotSupported
	addrTypeNotSupported
)

const (
	UserPassAuth    = uint8(2)
	userAuthVersion = uint8(1)
	authSuccess     = uint8(0)
	authFailure     = uint8(1)
)

type Sock5ModeServer struct {
	BaseServer
	listener net.Listener
}

// req
func (s *Sock5ModeServer) handleRequest(c net.Conn) {
	/*
		The SOCKS request is formed as follows:
		+----+-----+-------+------+----------+----------+
		|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
		+----+-----+-------+------+----------+----------+
		| 1  |  1  | X'00' |  1   | Variable |    2     |
		+----+-----+-------+------+----------+----------+
	*/
	header := make([]byte, 3)

	_, err := io.ReadFull(c, header)

	if err != nil {
		logs.Warn("illegal request", err)
		c.Close()
		return
	}

	switch header[1] {
	case connectMethod:
		s.handleConnect(c)
	case bindMethod:
		s.handleBind(c)
	case associateMethod:
		s.handleUDP(c)
	default:
		s.sendReply(c, commandNotSupported)
		c.Close()
	}
}

// reply
func (s *Sock5ModeServer) sendReply(c net.Conn, rep uint8) {
	reply := []byte{
		5,
		rep,
		0,
		1,
	}

	localAddr := c.LocalAddr().String()
	localHost, localPort, _ := net.SplitHostPort(localAddr)
	ipBytes := net.ParseIP(localHost).To4()
	nPort, _ := strconv.Atoi(localPort)
	reply = append(reply, ipBytes...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(nPort))
	reply = append(reply, portBytes...)

	c.Write(reply)
}

// do conn
func (s *Sock5ModeServer) doConnect(c net.Conn, command uint8) {
	addrType := make([]byte, 1)
	if _, err := io.ReadFull(c, addrType); err != nil {
		logs.Warn("read addr type error", err)
		s.sendReply(c, serverFailure)
		c.Close()
		return
	}
	var host string
	switch addrType[0] {
	case ipV4:
		ipv4 := make(net.IP, net.IPv4len)
		if _, err := io.ReadFull(c, ipv4); err != nil {
			logs.Warn("read ipv4 error", err)
			s.sendReply(c, serverFailure)
			c.Close()
			return
		}
		host = ipv4.String()
	case ipV6:
		ipv6 := make(net.IP, net.IPv6len)
		if _, err := io.ReadFull(c, ipv6); err != nil {
			logs.Warn("read ipv6 error", err)
			s.sendReply(c, serverFailure)
			c.Close()
			return
		}
		host = ipv6.String()
	case domainName:
		var domainLen uint8
		if err := binary.Read(c, binary.BigEndian, &domainLen); err != nil {
			logs.Warn("read domain len error", err)
			s.sendReply(c, serverFailure)
			c.Close()
			return
		}
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(c, domain); err != nil {
			logs.Warn("read domain error", err)
			s.sendReply(c, serverFailure)
			c.Close()
			return
		}
		host = string(domain)
	default:
		s.sendReply(c, addrTypeNotSupported)
		return
	}

	var port uint16
	if err := binary.Read(c, binary.BigEndian, &port); err != nil {
		logs.Warn("read port error", err)
		s.sendReply(c, serverFailure)
		c.Close()
		return
	}
	// connect to host
	addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
	var ltype string
	if command == associateMethod {
		ltype = common.CONN_UDP
	} else {
		ltype = common.CONN_TCP
	}
	s.DealClient(conn.NewConn(c), s.task.Client, addr, nil, ltype, func() {
		s.sendReply(c, succeeded)
	}, s.task.Flow, s.task.Target.LocalProxy, nil, nil)
	return
}

// conn
func (s *Sock5ModeServer) handleConnect(c net.Conn) {
	s.doConnect(c, connectMethod)
}

// passive mode
func (s *Sock5ModeServer) handleBind(c net.Conn) {
}
func (s *Sock5ModeServer) sendUdpReply(writeConn net.Conn, c net.Conn, rep uint8, serverIp string) {
	reply := []byte{
		5,
		rep,
		0,
		1,
	}
	localHost, localPort, _ := net.SplitHostPort(c.LocalAddr().String())
	localHost = serverIp
	ipBytes := net.ParseIP(localHost).To4()
	nPort, _ := strconv.Atoi(localPort)
	reply = append(reply, ipBytes...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(nPort))
	reply = append(reply, portBytes...)
	writeConn.Write(reply)

}

func (s *Sock5ModeServer) handleUDP(c net.Conn) {
	defer c.Close()
	addrType := make([]byte, 1)
	if _, err := io.ReadFull(c, addrType); err != nil {
		logs.Warn("read addr type error", err)
		s.sendReply(c, serverFailure)
		return
	}
	var host string
	switch addrType[0] {
	case ipV4:
		ipv4 := make(net.IP, net.IPv4len)
		if _, err := io.ReadFull(c, ipv4); err != nil {
			logs.Warn("read ipv4 error", err)
			s.sendReply(c, serverFailure)
			return
		}
		host = ipv4.String()
	case ipV6:
		ipv6 := make(net.IP, net.IPv6len)
		if _, err := io.ReadFull(c, ipv6); err != nil {
			logs.Warn("read ipv6 error", err)
			s.sendReply(c, serverFailure)
			return
		}
		host = ipv6.String()
	case domainName:
		var domainLen uint8
		if err := binary.Read(c, binary.BigEndian, &domainLen); err != nil {
			logs.Warn("read domain len error", err)
			s.sendReply(c, serverFailure)
			return
		}
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(c, domain); err != nil {
			logs.Warn("read domain error", err)
			s.sendReply(c, serverFailure)
			return
		}
		host = string(domain)
	default:
		s.sendReply(c, addrTypeNotSupported)
		return
	}
	//读取端口
	var port uint16
	if err := binary.Read(c, binary.BigEndian, &port); err != nil {
		logs.Warn("read port error", err)
		s.sendReply(c, serverFailure)
		return
	}
	logs.Warn(host, strconv.Itoa(int(port)))
	replyAddr, err := net.ResolveUDPAddr("udp", s.task.ServerIp+":0")
	if err != nil {
		logs.Error("build local reply addr error", err)
		return
	}
	reply, err := net.ListenUDP("udp", replyAddr)
	if err != nil {
		s.sendReply(c, addrTypeNotSupported)
		logs.Error("listen local reply udp port error")
		return
	}
	// reply the local addr
	s.sendUdpReply(c, reply, succeeded, common.GetServerIpByClientIp(c.RemoteAddr().(*net.TCPAddr).IP))
	defer reply.Close()
	// new a tunnel to client
	link := conn.NewLink("udp5", "", s.task.Client.Cnf.Crypt, s.task.Client.Cnf.Compress, c.RemoteAddr().String(), false, "")
	target, err := s.bridge.SendLinkInfo(s.task.Client.Id, link, s.task)
	if err != nil {
		logs.Warn("get connection from client id %d  error %s", s.task.Client.Id, err.Error())
		return
	}

	var clientAddr net.Addr
	// copy buffer
	go func() {
		b := common.BufPoolUdp.Get().([]byte)
		defer common.BufPoolUdp.Put(b)
		defer c.Close()

		for {
			n, laddr, err := reply.ReadFrom(b)
			if err != nil {
				logs.Error("read data from %s err %s", reply.LocalAddr().String(), err.Error())
				return
			}
			if clientAddr == nil {
				clientAddr = laddr
			}
			if _, err := target.Write(b[:n]); err != nil {
				logs.Error("write data to client error", err.Error())
				return
			}
		}
	}()

	go func() {
		var l int32
		b := common.BufPoolUdp.Get().([]byte)
		defer common.BufPoolUdp.Put(b)
		defer c.Close()
		for {
			if err := binary.Read(target, binary.LittleEndian, &l); err != nil || l >= common.PoolSizeUdp || l <= 0 {
				logs.Warn("read len bytes error", err.Error())
				return
			}
			binary.Read(target, binary.LittleEndian, b[:l])
			if err != nil {
				logs.Warn("read data form client error", err.Error())
				return
			}
			if _, err := reply.WriteTo(b[:l], clientAddr); err != nil {
				logs.Warn("write data to user ", err.Error())
				return
			}
		}
	}()

	b := common.BufPoolUdp.Get().([]byte)
	defer common.BufPoolUdp.Put(b)
	defer target.Close()
	for {
		_, err := c.Read(b)
		if err != nil {
			c.Close()
			return
		}
	}
}

// new conn
func (s *Sock5ModeServer) handleConn(c net.Conn) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(c, buf); err != nil {
		logs.Warn("negotiation err", err)
		c.Close()
		return
	}

	if version := buf[0]; version != 5 {
		logs.Warn("only support socks5, request from: ", c.RemoteAddr())
		c.Close()
		return
	}
	nMethods := int(buf[1])
	if nMethods == 0 {
		logs.Warn("socks5 client offered no auth methods, remote %s", c.RemoteAddr())
		c.Close()
		return
	}

	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(c, methods); err != nil {
		logs.Warn("wrong method")
		c.Close()
		return
	}

	needAuth := (s.task.Client.Cnf.U != "" && s.task.Client.Cnf.P != "") ||
		(s.task.MultiAccount != nil && len(s.task.MultiAccount.AccountMap) > 0)

	if needAuth {
		if !methodOffered(methods, UserPassAuth) {
			// Client cannot authenticate; reply no acceptable methods (RFC 1928).
			_, _ = c.Write([]byte{5, 0xFF})
			c.Close()
			logs.Warn("socks5 client %s does not support username/password auth", c.RemoteAddr())
			return
		}
		if _, err := c.Write([]byte{5, UserPassAuth}); err != nil {
			c.Close()
			return
		}
		if err := s.Auth(c); err != nil {
			c.Close()
			logs.Warn("Validation failed: %v, remote %s", err, c.RemoteAddr())
			return
		}
	} else {
		if _, err := c.Write([]byte{5, 0}); err != nil {
			c.Close()
			return
		}
	}
	s.handleRequest(c)
}

func methodOffered(methods []byte, method byte) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

// socks5 auth (RFC 1929 username/password)
func (s *Sock5ModeServer) Auth(c net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c, header); err != nil {
		return err
	}
	if header[0] != userAuthVersion {
		return errors.New("验证方式不被支持")
	}
	userLen := int(header[1])
	user := make([]byte, userLen)
	if userLen > 0 {
		if _, err := io.ReadFull(c, user); err != nil {
			return err
		}
	}
	plen := make([]byte, 1)
	if _, err := io.ReadFull(c, plen); err != nil {
		return errors.New("密码长度获取错误")
	}
	passLen := int(plen[0])
	pass := make([]byte, passLen)
	if passLen > 0 {
		if _, err := io.ReadFull(c, pass); err != nil {
			return err
		}
	}

	ok := false
	if s.task.MultiAccount != nil && len(s.task.MultiAccount.AccountMap) > 0 {
		// multi-user auth
		if expected, found := s.task.MultiAccount.AccountMap[string(user)]; found && string(pass) == expected {
			ok = true
		}
	} else {
		ok = string(user) == s.task.Client.Cnf.U && string(pass) == s.task.Client.Cnf.P
	}

	if ok {
		if _, err := c.Write([]byte{userAuthVersion, authSuccess}); err != nil {
			return err
		}
		return nil
	}
	if _, err := c.Write([]byte{userAuthVersion, authFailure}); err != nil {
		return err
	}
	return errors.New("验证不通过")
}

// start
func (s *Sock5ModeServer) Start() error {
	return conn.NewTcpListenerAndProcess(s.task.ServerIp+":"+strconv.Itoa(s.task.Port), func(c net.Conn) {
		if err := s.CheckFlowAndConnNum(s.task.Client); err != nil {
			logs.Warn("client id %d, task id %d, error %s, when socks5 connection", s.task.Client.Id, s.task.Id, err.Error())
			c.Close()
			return
		}
		logs.Trace("New socks5 connection,client %d,remote address %s", s.task.Client.Id, c.RemoteAddr())
		s.handleConn(c)
		s.task.Client.AddConn()
	}, &s.listener)
}

// new
func NewSock5ModeServer(bridge NetBridge, task *file.Tunnel) *Sock5ModeServer {
	s := new(Sock5ModeServer)
	s.bridge = bridge
	s.task = task
	return s
}

// close
func (s *Sock5ModeServer) Close() error {
	return s.listener.Close()
}
