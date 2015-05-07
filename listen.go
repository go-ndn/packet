package packet

import (
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

type entry struct {
	time time.Time
	ch   chan byte
	mu   sync.Mutex
}

type Listener struct {
	conn   net.PacketConn
	accept chan net.Conn
	closed chan struct{}
	m      map[string]*entry
	mu     sync.Mutex
}

func newListener(conn net.PacketConn) net.Listener {
	l := &Listener{
		conn:   conn,
		accept: make(chan net.Conn),
		closed: make(chan struct{}),
		m:      make(map[string]*entry),
	}
	go func() {
		b := make([]byte, packetSize)
		for {
			select {
			case <-l.closed:
				return
			default:
				conn.SetReadDeadline(time.Now().Add(Heartbeat))
				n, addr, err := conn.ReadFrom(b)
				if err != nil {
					continue
				}
				saddr := addr.String()

				l.mu.Lock()
				if _, ok := l.m[saddr]; !ok {
					l.m[saddr] = &entry{
						ch: make(chan byte, bufferSize),
					}
					c := newConn(l, conn.(net.Conn), addr)
					go func() {
						select {
						case <-l.closed:
						case l.accept <- c:
						}
					}()
				}
				ent := l.m[saddr]
				l.mu.Unlock()
				if cap(ent.ch)-len(ent.ch) < n {
					continue
				}
				ent.mu.Lock()
				ent.time = time.Now()
				ent.mu.Unlock()
				for _, bb := range b[:n] {
					ent.ch <- bb
				}
			}
		}
	}()
	return l
}

func (l *Listener) read(b []byte, addr net.Addr) (n int, err error) {
	saddr := addr.String()
	l.mu.Lock()
	ent, ok := l.m[saddr]
	l.mu.Unlock()
	if !ok {
		err = io.EOF
		return
	}
	for n = 0; n < len(b); n++ {
		select {
		case b[n] = <-ent.ch:
		case <-time.After(Dead):
			ent.mu.Lock()
			ok := time.Now().Sub(ent.time) < Dead
			ent.mu.Unlock()
			if n == 0 && !ok {
				l.mu.Lock()
				delete(l.m, saddr)
				l.mu.Unlock()

				err = io.EOF
			}
			return
		}
	}
	return
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
