package conn

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

type bytesThenErrConn struct {
	net.Conn
	payload []byte
	err     error
	read    bool
}

func (c *bytesThenErrConn) Read(b []byte) (int, error) {
	if c.read {
		return 0, c.err
	}
	c.read = true
	return copy(b, c.payload), c.err
}

func TestGetHostReturnsBufferedBytesOnReadError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	payload := []byte("SSH-2.0-OpenSSH_9.0\r\n")
	if err := serverConn.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	type result struct {
		rb  []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		_, _, rb, err, _ := NewConn(serverConn).GetHost()
		done <- result{rb: rb, err: err}
	}()

	if _, err := clientConn.Write(payload); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		if got.err == nil {
			t.Fatal("expected GetHost to fail for non-HTTP payload")
		}
		if !bytes.Equal(got.rb, payload) {
			t.Fatalf("expected buffered payload %q, got %q", payload, got.rb)
		}
	case <-time.After(time.Second):
		t.Fatal("GetHost did not return after read deadline")
	}
}

func TestGetHostReturnsBufferedBytesWhenReadReturnsBytesAndError(t *testing.T) {
	payload := []byte("SSH-2.0-OpenSSH_9.0\r\n")
	c := &bytesThenErrConn{
		payload: payload,
		err:     io.ErrUnexpectedEOF,
	}

	_, _, rb, err, _ := NewConn(c).GetHost()
	if err == nil {
		t.Fatal("expected GetHost to fail for non-HTTP payload")
	}
	if !bytes.Equal(rb, payload) {
		t.Fatalf("expected buffered payload %q, got %q", payload, rb)
	}
}
