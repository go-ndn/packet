package packet

import (
	"io"
	"net"
	"sync"
	"syscall"

	"github.com/go-ndn/tlv"
)

type listener struct {
	net.PacketConn
	accept     chan net.Conn
	closed     chan struct{}
	open       map[string]io.ReadWriter
	sync.Mutex // open
}

func newListener(conn net.PacketConn) net.Listener {
	l := &listener{
		PacketConn: conn,
		accept:     make(chan net.Conn),
		closed:     make(chan struct{}),
		open:       make(map[string]io.ReadWriter),
	}
	go func() {
		b := make([]byte, tlv.MaxSize)
		for {
			n, raddr, err := conn.ReadFrom(b)
			if err != nil {
				return
			}
			addr := raddr.String()

			l.Lock()
			buf, ok := l.open[addr]
			if !ok {
				buf = newBuffer()
				l.open[addr] = buf
			}
			l.Unlock()

			buf.Write(b[:n])

			if !ok {
				// new connection
				go func() {
					c := newConn(buf, writerFunc(func(b []byte) (int, error) {
						return conn.WriteTo(b, raddr)
					}), closerFunc(func() error { return nil }), conn.LocalAddr(), raddr)
					select {
					case <-l.closed:
						return
					case l.accept <- c:
					}
					select {
					case <-l.closed:
					case <-c.closed:
						l.Lock()
						delete(l.open, addr)
						l.Unlock()
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

func listenUDP(network, addr string) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, err
	}
	c, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, err
	}
	return newListener(c), nil
}

// Listen announces on the local network address.
//
// If the network is not packet-oriented, it calls net.Listen directly.
func Listen(network, address string) (net.Listener, error) {
	switch network {
	case "udp", "udp4", "udp6":
		return listenUDP(network, address)
	default:
		return net.Listen(network, address)
	}
}
