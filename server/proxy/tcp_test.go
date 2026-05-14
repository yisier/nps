package proxy

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
)

type tunnelTestBridge struct {
	target net.Conn
}

func (b *tunnelTestBridge) SendLinkInfo(clientId int, link *conn.Link, t *file.Tunnel) (net.Conn, error) {
	return b.target, nil
}

func TestProcessTunnelForwardsBufferedBytesWhenAuthProbeFails(t *testing.T) {
	confPath := t.TempDir()
	confDir := filepath.Join(confPath, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"clients.json", "tasks.json", "hosts.json"} {
		if err := os.WriteFile(filepath.Join(confDir, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	oldConfPath := common.ConfPath
	common.ConfPath = confPath
	t.Cleanup(func() {
		common.ConfPath = oldConfPath
	})

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	targetClient, targetServer := net.Pipe()
	defer targetClient.Close()
	defer targetServer.Close()

	client := file.NewClient("test", true, true)
	client.Id = 1
	client.Cnf.U = "user"
	client.Cnf.P = "pass"
	task := &file.Tunnel{
		Id:     1,
		Client: client,
		Target: &file.Target{
			TargetStr: "127.0.0.1:22",
		},
	}
	server := NewTunnelModeServer(ProcessTunnel, &tunnelTestBridge{target: targetServer}, task)

	done := make(chan error, 1)
	go func() {
		done <- ProcessTunnel(conn.NewConn(serverConn), server)
	}()

	payload := []byte("SSH-2.0-OpenSSH_9.0\r\n")
	if _, err := clientConn.Write(payload); err != nil {
		t.Fatal(err)
	}

	if err := targetClient.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(targetClient, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("expected forwarded payload %q, got %q", payload, got)
	}

	clientConn.Close()
	targetClient.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessTunnel did not exit after connections closed")
	}
}
