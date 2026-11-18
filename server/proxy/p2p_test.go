package proxy

import (
	"net"
	"strconv"
	"sync"
	"testing"

	"ehang.io/nps/lib/common"
)

// TestHandleP2PConcurrent 覆盖 #227：P2P 报文由独立协程处理，
// handleP2P 并发读写 s.p2p 时不能触发 concurrent map writes / data race。
func TestHandleP2PConcurrent(t *testing.T) {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	server := NewP2PServer(0)
	server.listener = listener

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:23456")
	if err != nil {
		t.Fatal(err)
	}

	const count = 100
	var wg sync.WaitGroup

	// provider 先上线，避免 visitor 走 20 秒的等待分支
	for i := 0; i < count; i++ {
		key := "key" + strconv.Itoa(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			server.handleP2P(addr, key+common.CONN_DATA_SEQ+common.WORK_P2P_PROVIDER)
		}()
	}
	wg.Wait()

	for i := 0; i < count; i++ {
		key := "key" + strconv.Itoa(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			server.handleP2P(addr, key+common.CONN_DATA_SEQ+common.WORK_P2P_VISITOR)
		}()
	}
	wg.Wait()

	server.mu.Lock()
	left := len(server.p2p)
	server.mu.Unlock()
	if left != 0 {
		t.Fatalf("expected all p2p pairs to be cleaned up, got %d left", left)
	}
}

func TestHandleP2PIgnoresMalformedPacket(t *testing.T) {
	server := NewP2PServer(0)
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:23457")
	if err != nil {
		t.Fatal(err)
	}
	server.handleP2P(addr, "no-separator")
	if len(server.p2p) != 0 {
		t.Fatalf("malformed packet should be ignored")
	}
}
