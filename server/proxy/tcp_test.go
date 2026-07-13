package proxy

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func setupProxyTestConf(t *testing.T) {
	t.Helper()
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
}

func newHTTPProxyServer(t *testing.T, u, p string, target net.Conn) *TunnelModeServer {
	t.Helper()
	client := file.NewClient("test", true, true)
	client.Id = 1
	client.Cnf.U = u
	client.Cnf.P = p
	task := &file.Tunnel{
		Id:     1,
		Client: client,
		Mode:   "httpProxy",
		Target: &file.Target{TargetStr: "127.0.0.1:80"},
	}
	return NewTunnelModeServer(ProcessHttp, &tunnelTestBridge{target: target}, task)
}

func TestProcessHttpConnectAuthBeforeTunnel(t *testing.T) {
	setupProxyTestConf(t)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// target unused: auth must fail before DealClient
	targetClient, targetServer := net.Pipe()
	defer targetClient.Close()
	defer targetServer.Close()

	server := newHTTPProxyServer(t, "user", "pass", targetServer)

	done := make(chan error, 1)
	go func() {
		done <- ProcessHttp(conn.NewConn(serverConn), server)
	}()

	// CONNECT without credentials
	req := "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n"
	if _, err := clientConn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	_ = clientConn.SetReadDeadline(time.Now().Add(time.Second))
	br := bufio.NewReader(clientConn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusProxyAuthRequired {
		t.Fatalf("expected 407 before CONNECT tunnel, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Proxy-Authenticate"); !strings.Contains(got, "Basic") {
		t.Fatalf("expected Proxy-Authenticate Basic challenge, got %q", got)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected auth error")
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessHttp did not exit after auth failure")
	}
}

func TestProcessHttpConnectWithProxyAuth(t *testing.T) {
	setupProxyTestConf(t)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	targetClient, targetServer := net.Pipe()
	defer targetClient.Close()
	defer targetServer.Close()

	server := newHTTPProxyServer(t, "user", "pass", targetServer)

	done := make(chan error, 1)
	go func() {
		done <- ProcessHttp(conn.NewConn(serverConn), server)
	}()

	token := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	req := "CONNECT example.com:443 HTTP/1.1\r\n" +
		"Host: example.com:443\r\n" +
		"Proxy-Authorization: Basic " + token + "\r\n\r\n"
	if _, err := clientConn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	_ = clientConn.SetReadDeadline(time.Now().Add(time.Second))
	br := bufio.NewReader(clientConn)
	// First line of CONNECT success is not always a full http.Response parse-friendly body;
	// read status line.
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("expected 200 Connection established after auth, got %q", statusLine)
	}
	// consume remaining headers
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	// After tunnel is up, write application data and ensure it reaches target.
	payload := []byte("hello-tls")
	if _, err := clientConn.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = targetClient.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(targetClient, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("expected %q, got %q", payload, got)
	}

	clientConn.Close()
	targetClient.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ProcessHttp did not exit")
	}
}

func TestProcessHttpPlainGetWithProxyAuth(t *testing.T) {
	setupProxyTestConf(t)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	targetClient, targetServer := net.Pipe()
	defer targetClient.Close()
	defer targetServer.Close()

	server := newHTTPProxyServer(t, "user", "pass", targetServer)

	done := make(chan error, 1)
	go func() {
		done <- ProcessHttp(conn.NewConn(serverConn), server)
	}()

	token := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	req := "GET http://example.com/ HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Proxy-Authorization: Basic " + token + "\r\n\r\n"
	if _, err := clientConn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	// Absolute-form request is forwarded (buffered) to the tunnel target.
	_ = targetClient.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 512)
	n, err := targetClient.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf[:n], []byte("GET ")) {
		t.Fatalf("expected forwarded GET, got %q", buf[:n])
	}

	clientConn.Close()
	targetClient.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ProcessHttp did not exit")
	}
}
