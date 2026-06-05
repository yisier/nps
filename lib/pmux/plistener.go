package pmux

import (
	"errors"
	"net"
)

type PortListener struct {
	net.Listener
	connCh  chan *PortConn
	addr    net.Addr
	isClose bool
	done    chan struct{}
}

func NewPortListener(connCh chan *PortConn, addr net.Addr) *PortListener {
	return &PortListener{
		connCh: connCh,
		addr:   addr,
		done:   make(chan struct{}),
	}
}

func (pListener *PortListener) Accept() (net.Conn, error) {
	if pListener.isClose {
		return nil, errors.New("the listener has closed")
	}
	select {
	case <-pListener.done:
		return nil, errors.New("the listener has closed")
	case conn := <-pListener.connCh:
		if conn != nil {
			return conn, nil
		}
		return nil, errors.New("the listener has closed")
	}
}

func (pListener *PortListener) Close() error {
	if pListener.isClose {
		return errors.New("the listener has closed")
	}
	pListener.isClose = true
	close(pListener.done)
	return nil
}

func (pListener *PortListener) Addr() net.Addr {
	return pListener.addr
}
