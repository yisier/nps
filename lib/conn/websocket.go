package conn

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket连接包装器
type WebSocketConn struct {
	conn   *websocket.Conn
	buffer []byte // 缓冲区用于存储未读完的数据
}

func (w *WebSocketConn) Read(b []byte) (n int, err error) {
	// 如果缓冲区有数据，先从缓冲区读取
	if len(w.buffer) > 0 {
		n = copy(b, w.buffer)
		w.buffer = w.buffer[n:]
		return n, nil
	}

	// 缓冲区为空，从WebSocket读取新消息
	_, data, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	// 将数据复制到输出缓冲区
	n = copy(b, data)

	// 如果数据长度超过输出缓冲区，将剩余数据存储到内部缓冲区
	if len(data) > len(b) {
		w.buffer = make([]byte, len(data)-len(b))
		copy(w.buffer, data[len(b):])
	}

	return n, nil
}

func (w *WebSocketConn) Write(b []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *WebSocketConn) Close() error {
	return w.conn.Close()
}

func (w *WebSocketConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *WebSocketConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *WebSocketConn) SetDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WebSocketConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

// 创建WebSocket客户端连接
func NewWebSocketConn(serverAddr string, tlsEnable bool) (net.Conn, error) {
	var scheme string
	if tlsEnable {
		scheme = "wss"
	} else {
		scheme = "ws"
	}

	u := url.URL{Scheme: scheme, Host: serverAddr, Path: "/ws"}
	log.Printf("[DEBUG] Attempting WebSocket connection to: %s", u.String())

	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	if tlsEnable {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	c, resp, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("[DEBUG] WebSocket dial failed: %v", err)
		if resp != nil {
			log.Printf("[DEBUG] HTTP response status: %s", resp.Status)
		}
		return nil, err
	}

	log.Printf("[DEBUG] WebSocket connection established successfully")
	return &WebSocketConn{conn: c}, nil
}

// 创建WebSocket监听器
func NewWebSocketListener(addr string, tlsConfig *tls.Config) (net.Listener, error) {
	return &WebSocketListener{
		addr:      addr,
		tlsConfig: tlsConfig,
		conns:     make(chan net.Conn, 100),
		closed:    make(chan struct{}),
	}, nil
}

type WebSocketListener struct {
	addr      string
	tlsConfig *tls.Config
	conns     chan net.Conn
	closed    chan struct{}
	server    *http.Server
}

func (l *WebSocketListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
		return nil, &net.OpError{Op: "accept", Net: "websocket", Err: net.ErrClosed}
	}
}

func (l *WebSocketListener) Close() error {
	close(l.closed)
	if l.server != nil {
		return l.server.Close()
	}
	return nil
}

func (l *WebSocketListener) Addr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", l.addr)
	return addr
}

func (l *WebSocketListener) Start() error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[DEBUG] WebSocket upgrade request from %s", r.RemoteAddr)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[DEBUG] WebSocket upgrade failed: %v", err)
			return
		}

		log.Printf("[DEBUG] WebSocket connection upgraded successfully from %s", r.RemoteAddr)
		select {
		case l.conns <- &WebSocketConn{conn: conn}:
			log.Printf("[DEBUG] WebSocket connection added to channel")
		case <-l.closed:
			log.Printf("[DEBUG] WebSocket listener closed, closing connection")
			conn.Close()
		}
	})

	l.server = &http.Server{
		Addr:    l.addr,
		Handler: mux,
	}

	log.Printf("[DEBUG] Starting WebSocket server on %s", l.addr)
	if l.tlsConfig != nil {
		l.server.TLSConfig = l.tlsConfig
		return l.server.ListenAndServeTLS("", "")
	}
	return l.server.ListenAndServe()
}
