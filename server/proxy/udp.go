package proxy

import (
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ehang.io/nps/bridge"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"github.com/astaxie/beego/logs"
)

const (
	udpSessionIdleTimeout = 120 * time.Second
	udpSweepInterval      = 30 * time.Second
	udpReadDeadline       = 60 * time.Second
	udpBuildTimeout       = 5 * time.Second
)

// udpSession 代表一个客户端 src addr ↔ npc 之间的 UDP 转发会话。
//
// 同一个 src addr 在并发场景下可能被多个 goroutine 同时尝试建立会话。为避免
// 重复建立，采用"原子占位"模式：第一个 goroutine 通过 sync.Map.LoadOrStore
// 占位（此时 ready 通道未关闭），后续 goroutine 检测到占位后阻塞在 ready
// 通道上等待会话就绪，然后复用同一个 target 转发数据。这样：
//   - 同一 src addr 始终只有 1 条到 npc 的 mux stream
//   - 只有"赢家"消耗一个 NowConn 配额，输家不再重复占用
//   - 避免了 race window 期间的 NowConn 配额泄漏
type udpSession struct {
	target     io.ReadWriteCloser // 用于 Read/Write 的封装层（可能含加密/压缩）
	rawConn    net.Conn           // 底层 mux conn，用于 SetReadDeadline
	lastActive int64              // 最近活跃时间（unix nano，原子读写）
	ready      chan struct{}      // 会话就绪后关闭；建立失败时也关闭
	err        error              // 建立失败时设置；ready 关闭后才允许读
}

func (u *udpSession) touch() {
	atomic.StoreInt64(&u.lastActive, time.Now().UnixNano())
}

type UdpModeServer struct {
	BaseServer
	addrMap   sync.Map
	listener  *net.UDPConn
	closeOnce sync.Once
	closeCh   chan struct{}
}

func NewUdpModeServer(bridge *bridge.Bridge, task *file.Tunnel) *UdpModeServer {
	s := new(UdpModeServer)
	s.bridge = bridge
	s.task = task
	s.closeCh = make(chan struct{})
	return s
}

// Start 启动 UDP 监听，主循环只负责快速收包并分发，不做任何耗时操作。
func (s *UdpModeServer) Start() error {
	var err error
	if s.task.ServerIp == "" {
		s.task.ServerIp = "0.0.0.0"
	}
	s.listener, err = net.ListenUDP("udp", &net.UDPAddr{net.ParseIP(s.task.ServerIp), s.task.Port, ""})
	if err != nil {
		return err
	}
	go s.sweeper()
	for {
		buf := common.BufPoolUdp.Get().([]byte)
		n, addr, err := s.listener.ReadFromUDP(buf)
		if err != nil {
			common.BufPoolUdp.Put(buf)
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			continue
		}

		if IsGlobalBlackIp(addr.String()) {
			common.BufPoolUdp.Put(buf)
			continue
		}
		if common.IsBlackIp(addr.String(), s.task.Client.VerifyKey, s.task.Client.BlackIpList) {
			common.BufPoolUdp.Put(buf)
			continue
		}

		go s.process(addr, buf, n)
	}
	return nil
}

// process 处理单个 UDP 包。函数持有 buf 的所有权，所有路径必须归还。
func (s *UdpModeServer) process(addr *net.UDPAddr, buf []byte, n int) {
	key := addr.String()
	data := buf[:n]

	// 快路径：会话已存在
	if v, ok := s.addrMap.Load(key); ok {
		s.dispatch(key, v.(*udpSession), data, n)
		common.BufPoolUdp.Put(buf)
		return
	}

	// 慢路径：尝试成为该 key 的"会话建立者"
	placeholder := &udpSession{ready: make(chan struct{})}
	placeholder.touch()
	if existing, loaded := s.addrMap.LoadOrStore(key, placeholder); loaded {
		// 输了占位竞争，等赢家把会话建好后复用
		s.dispatch(key, existing.(*udpSession), data, n)
		common.BufPoolUdp.Put(buf)
		return
	}

	// 赢得占位 —— 我去建立会话
	s.runSession(addr, key, placeholder, buf, n)
}

// dispatch 把数据写入 sess.target。若 sess 仍在建立中则阻塞等待，超时则丢包。
func (s *UdpModeServer) dispatch(key string, sess *udpSession, data []byte, n int) {
	if sess.ready != nil {
		select {
		case <-sess.ready:
			if sess.err != nil {
				logs.Trace("udp session build failed for %s: %v", key, sess.err)
				return
			}
		case <-time.After(udpBuildTimeout):
			logs.Warn("udp session build timeout for %s, drop packet", key)
			return
		case <-s.closeCh:
			return
		}
	}
	if _, err := sess.target.Write(data); err != nil {
		logs.Warn(err)
		s.removeSession(key, sess)
		return
	}
	sess.touch()
	s.task.Client.Flow.Add(int64(n), int64(n))
}

// runSession 由占位赢家执行：建立到 npc 的 stream、发送首包、运行下行读循环。
// buf 由本函数负责归还。
func (s *UdpModeServer) runSession(addr *net.UDPAddr, key string, sess *udpSession, buf []byte, n int) {
	data := buf[:n]

	// 失败时统一清理：关 ready 通道唤醒所有输家、删占位、归还 buf。
	failBuild := func(err error) {
		sess.err = err
		close(sess.ready)
		s.addrMap.Delete(key)
		common.BufPoolUdp.Put(buf)
	}

	if err := s.CheckFlowAndConnNum(s.task.Client); err != nil {
		logs.Warn("client id %d, task id %d,error %s, when udp connection", s.task.Client.Id, s.task.Id, err.Error())
		failBuild(err)
		return
	}
	// 只有赢家消耗 NowConn 配额，函数返回时释放。
	defer s.task.Client.AddConn()

	link := conn.NewLink(common.CONN_UDP, s.task.Target.TargetStr, s.task.Client.Cnf.Crypt, s.task.Client.Cnf.Compress, addr.String(), s.task.Target.LocalProxy, "")
	clientConn, err := s.bridge.SendLinkInfo(s.task.Client.Id, link, s.task)
	if err != nil {
		failBuild(err)
		return
	}

	target := conn.GetConn(clientConn, s.task.Client.Cnf.Crypt, s.task.Client.Cnf.Compress, nil, true)
	sess.target = target
	sess.rawConn = clientConn
	sess.touch()
	close(sess.ready) // 唤醒所有等待该会话的输家

	defer s.removeSession(key, sess)

	logs.Trace("New udp connection,client %d,remote address %s", s.task.Client.Id, addr)

	if _, err := target.Write(data); err != nil {
		logs.Warn(err)
		common.BufPoolUdp.Put(buf)
		return
	}
	common.BufPoolUdp.Put(buf)
	s.task.Client.Flow.Add(int64(n), int64(n))

	// 下行读循环
	rbuf := common.BufPoolUdp.Get().([]byte)
	defer common.BufPoolUdp.Put(rbuf)

	for {
		// 设到 rawConn 上：mux stream 的 SetReadDeadline 是底层 net.Conn 实现，
		// 比设在加密/压缩包装层 target 上更可靠。
		clientConn.SetReadDeadline(time.Now().Add(udpReadDeadline))
		rn, err := target.Read(rbuf)
		if err != nil {
			// sweeper 主动 Close、idle deadline 触发、或对端断开都会落到这里
			return
		}
		sess.touch()
		if _, err := s.listener.WriteTo(rbuf[:rn], addr); err != nil {
			logs.Warn(err)
			return
		}
		s.task.Client.Flow.Add(int64(rn), int64(rn))
	}
}

// removeSession 安全地从 addrMap 移除并关闭会话。
// 用 == 比对避免误删被替换的新 session（虽然当前协议不会发生，但保持防御性）。
func (s *UdpModeServer) removeSession(key string, sess *udpSession) {
	if v, ok := s.addrMap.Load(key); ok && v.(*udpSession) == sess {
		s.addrMap.Delete(key)
	}
	if sess.target != nil {
		sess.target.Close()
	}
}

// sweeper 周期性扫描 addrMap，把空闲超过 udpSessionIdleTimeout 的会话清掉。
// 关闭 target 会让阻塞中的 target.Read 立即返回错误，让对应的 runSession
// 走 defer 链路自然退出（释放 NowConn 配额、删 addrMap 条目）。
func (s *UdpModeServer) sweeper() {
	ticker := time.NewTicker(udpSweepInterval)
	defer ticker.Stop()
	idleNs := int64(udpSessionIdleTimeout)
	for {
		select {
		case <-s.closeCh:
			return
		case <-ticker.C:
			now := time.Now().UnixNano()
			s.addrMap.Range(func(k, v interface{}) bool {
				sess := v.(*udpSession)
				// 跳过仍在建立中的占位（target 尚未填充）
				if sess.target == nil {
					return true
				}
				if now-atomic.LoadInt64(&sess.lastActive) > idleNs {
					s.removeSession(k.(string), sess)
				}
				return true
			})
		}
	}
}

func (s *UdpModeServer) Close() error {
	s.closeOnce.Do(func() {
		close(s.closeCh)
	})
	s.addrMap.Range(func(k, v interface{}) bool {
		s.removeSession(k.(string), v.(*udpSession))
		return true
	})
	return s.listener.Close()
}
