package proxy

import (
	"net"
	"strings"
	"sync"
	"time"

	"ehang.io/nps/lib/common"
	"github.com/astaxie/beego/logs"
)

type P2PServer struct {
	BaseServer
	p2pPort  int
	mu       sync.Mutex
	p2p      map[string]*p2p
	listener *net.UDPConn
}

type p2p struct {
	visitorAddr  *net.UDPAddr
	providerAddr *net.UDPAddr
}

func NewP2PServer(p2pPort int) *P2PServer {
	return &P2PServer{
		p2pPort: p2pPort,
		p2p:     make(map[string]*p2p),
	}
}

func (s *P2PServer) Start() error {
	logs.Info("start p2p server port", s.p2pPort)
	var err error
	s.listener, err = net.ListenUDP("udp", &net.UDPAddr{net.ParseIP("0.0.0.0"), s.p2pPort, ""})
	if err != nil {
		return err
	}
	for {
		buf := common.BufPoolUdp.Get().([]byte)
		n, addr, err := s.listener.ReadFromUDP(buf)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			continue
		}
		go s.handleP2P(addr, string(buf[:n]))
	}
	return nil
}

func (s *P2PServer) handleP2P(addr *net.UDPAddr, str string) {
	arr := strings.Split(str, common.CONN_DATA_SEQ)
	if len(arr) < 2 {
		return
	}
	key, role := arr[0], arr[1]
	logs.Trace("new p2p connection ,role %s , password %s ,local address %s", role, key, addr.String())

	// 每条 UDP 报文都在独立协程中处理，s.p2p 必须加锁访问。
	// 并发写 map 是 fatal error，无法 recover，会直接终止 nps 进程（#227）。
	s.mu.Lock()
	v, ok := s.p2p[key]
	if !ok {
		v = new(p2p)
		s.p2p[key] = v
	}
	if role == common.WORK_P2P_VISITOR {
		v.visitorAddr = addr
	} else {
		v.providerAddr = addr
	}
	s.mu.Unlock()

	if role != common.WORK_P2P_VISITOR {
		return
	}

	// 等待 provider 上线，最多 20 秒。等待期间不持锁，避免阻塞其他连接。
	var providerAddr *net.UDPAddr
	for i := 20; i > 0; i-- {
		s.mu.Lock()
		providerAddr = v.providerAddr
		s.mu.Unlock()
		if providerAddr != nil {
			break
		}
		time.Sleep(time.Second)
	}
	if providerAddr != nil {
		s.listener.WriteTo([]byte(providerAddr.String()), addr)
		s.listener.WriteTo([]byte(addr.String()), providerAddr)
	}

	s.mu.Lock()
	delete(s.p2p, key)
	s.mu.Unlock()
}
