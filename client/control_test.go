package client

import (
	"testing"

	"ehang.io/nps/lib/common"
)

func TestGetNextAddr(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"1.2.3.4:8029", 1, "1.2.3.4:8030"},
		{"1.2.3.4:8029", 2, "1.2.3.4:8031"},
		{"[::1]:8029", 1, "[::1]:8030"},
	}
	for _, c := range cases {
		got, err := getNextAddr(c.in, c.n)
		if err != nil {
			t.Fatalf("getNextAddr(%q, %d) error: %v", c.in, c.n, err)
		}
		if got != c.want {
			t.Fatalf("getNextAddr(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}

	for _, invalid := range []string{"1.2.3.4", "1.2.3.4:", "1.2.3.4:x", "a:b:c"} {
		if _, err := getNextAddr(invalid, 1); err == nil {
			t.Fatalf("getNextAddr(%q) should fail", invalid)
		}
	}
}

func TestGetAddrInterval(t *testing.T) {
	if v, err := getAddrInterval("1.2.3.4:8029", "1.2.3.4:8030", "1.2.3.4:8031"); err != nil || v != 1 {
		t.Fatalf("unexpected interval %d, err %v", v, err)
	}
	if v, err := getAddrInterval("1.2.3.4:8031", "1.2.3.4:8030", "1.2.3.4:8029"); err != nil || v != -1 {
		t.Fatalf("unexpected interval %d, err %v", v, err)
	}
	if v, err := getAddrInterval("[::1]:8029", "[::1]:8030", "[::1]:8031"); err != nil || v != 1 {
		t.Fatalf("unexpected interval %d, err %v", v, err)
	}
}

// 覆盖 #126：服务端未配置 p2p_ip / p2p_port 时会返回空地址或 ":0"，
// 客户端应给出明确错误而不是 strconv.Atoi 之类的内部错误。
func TestHandleP2PUdpRejectsInvalidServerAddr(t *testing.T) {
	for _, addr := range []string{"", ":0", "1.2.3.4:", "0.0.0.0:"} {
		if _, _, err := handleP2PUdp("127.0.0.1:0", addr, "pwd", common.WORK_P2P_VISITOR); err == nil {
			t.Fatalf("expected an error for p2p server address %q", addr)
		}
	}
}
