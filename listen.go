package packet

import (
	"net"
	"syscall"
	"time"
)

type Listener struct {
	conn   net.PacketConn
	accept chan net.Conn
	closed chan struct{}
	r      *reader
}

func newListener(conn net.PacketConn) net.Listener {
	l := &Listener{
		conn:   conn,
		accept: make(chan net.Conn),
		closed: make(chan struct{}),
		r:      newReader(),
	}
	go func() {
		b := make([]byte, packetSize)
		for {
			select {
			case <-l.closed:
				return
			default:
				conn.SetReadDeadline(time.Now().Add(Dead))
				n, addr, err := conn.ReadFrom(b)
				if err != nil {
					continue
				}
				if l.r.write(b[:n], addr) {
					c := newConn(l.r, conn.(net.Conn), addr)
					go func() {
						select {
						case <-l.closed:
						case l.accept <- c:
						}
					}()
				}
			}
		}
	}()
	return l
}

func (l *Listener) Accept() (net.Conn, error) {
	c, ok := <-l.accept
	if !ok {
		return nil, syscall.EINVAL
	}
	return c, nil
}

func (l *Listener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func (l *Listener) Close() error {
	close(l.accept)
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
