package packet

import (
	"net"
	"syscall"
)

type listener struct {
	net.PacketConn
	accept chan net.Conn
	closed chan struct{}
}

func newListener(conn net.PacketConn) net.Listener {
	l := &listener{
		PacketConn: conn,
		accept:     make(chan net.Conn),
		closed:     make(chan struct{}),
	}
	go func() {
		buf := newBuffer()
		b := make([]byte, packetSize)
		for {
			n, addr, err := conn.ReadFrom(b)
			if err != nil {
				return
			}
			// write to buffer
			if buf.WriteTo(addr.String(), b[:n]) {
				// new connection
				go func() {
					select {
					case <-l.closed:
					case l.accept <- newConn(buf, conn.(net.Conn), addr):
					}
				}()
			}
		}
	}()
	return l
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		// already closed
		return nil, syscall.EINVAL
	case c := <-l.accept:
		// accept new connection
		return c, nil
	}
}

func (l *listener) Addr() net.Addr {
	return l.PacketConn.LocalAddr()
}

func (l *listener) Close() error {
	close(l.closed)
	return l.PacketConn.Close()
}

func Listen(network, addr string) (l net.Listener, err error) {
	c, err := net.ListenPacket(network, addr)
	if err != nil {
		return
	}
	l = newListener(c)
	return
}
