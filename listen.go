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
			n, raddr, err := conn.ReadFrom(b)
			if err != nil {
				return
			}
			// write to buffer
			if buf.WriteTo(raddr.String(), b[:n]) {
				// new connection
				go func() {
					select {
					case <-l.closed:
					case l.accept <- newConn(buf, conn.(net.Conn), raddr):
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

func listen(network, addr string) (net.Listener, error) {
	c, err := net.ListenPacket(network, addr)
	if err != nil {
		return nil, err
	}
	return newListener(c), nil
}

func Listen(network, address string) (net.Listener, error) {
	switch network {
	case "udp", "udp4", "udp6", "ip", "ip4", "ip6", "unixgram":
		return listen(network, address)
	default:
		return net.Listen(network, address)
	}
}
