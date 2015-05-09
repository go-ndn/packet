package packet

import (
	"net"
	"syscall"
)

type Listener struct {
	conn   net.PacketConn
	accept chan net.Conn
	closed chan struct{}
}

func newListener(conn net.PacketConn) net.Listener {
	l := &Listener{
		conn:   conn,
		accept: make(chan net.Conn),
		closed: make(chan struct{}),
	}
	go func() {
		r := newReader()
		b := make([]byte, packetSize)
		for {
			n, addr, err := conn.ReadFrom(b)
			if err != nil {
				return
			}
			if r.write(b[:n], addr.String()) {
				c := newConn(r, conn.(net.Conn), addr)
				go func() {
					select {
					case <-l.closed:
					case l.accept <- c:
					}
				}()
			}
		}
	}()
	return l
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, syscall.EINVAL
	case c := <-l.accept:
		return c, nil
	}
}

func (l *Listener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func (l *Listener) Close() error {
	close(l.closed)
	return l.conn.Close()
}

func Listen(network, addr string) (l net.Listener, err error) {
	c, err := net.ListenPacket(network, addr)
	if err != nil {
		return
	}
	l = newListener(c)
	return
}
