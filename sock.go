package packet

import (
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

const (
	Heartbeat  = 10 * time.Second
	Dead       = 2 * Heartbeat
	packetSize = 8192
	bufferSize = 131072
)

type entry struct {
	time time.Time
	ch   chan byte
	mu   sync.Mutex
}

type reader struct {
	m  map[string]*entry
	mu sync.Mutex
}

func newReader() *reader {
	return &reader{m: make(map[string]*entry)}
}

func (r *reader) read(b []byte, addr net.Addr) (n int, err error) {
	var saddr string
	if addr != nil {
		saddr = addr.String()
	}
	r.mu.Lock()
	ent, ok := r.m[saddr]
	r.mu.Unlock()
	if !ok {
		err = io.EOF
		return
	}
	for n = 0; n < len(b); n++ {
		select {
		case b[n] = <-ent.ch:
		case <-time.After(Dead):
			ent.mu.Lock()
			ok := time.Since(ent.time) < Dead
			ent.mu.Unlock()
			if n == 0 && !ok {
				r.mu.Lock()
				delete(r.m, saddr)
				r.mu.Unlock()

				err = io.EOF
			}
			return
		}
	}
	return
}

func (r *reader) write(b []byte, addr net.Addr) (create bool) {
	var saddr string
	if addr != nil {
		saddr = addr.String()
	}
	r.mu.Lock()
	if _, ok := r.m[saddr]; !ok {
		r.m[saddr] = &entry{
			ch: make(chan byte, bufferSize),
		}
		create = true
	}
	ent := r.m[saddr]
	r.mu.Unlock()
	if cap(ent.ch)-len(ent.ch) < len(b) {
		return
	}
	ent.mu.Lock()
	ent.time = time.Now()
	ent.mu.Unlock()
	for _, bb := range b {
		ent.ch <- bb
	}
	return
}

type Conn struct {
	r      *reader
	conn   net.Conn
	raddr  net.Addr
	closed chan struct{}
}

func newConn(r *reader, conn net.Conn, raddr net.Addr) *Conn {
	c := &Conn{
		r:      r,
		conn:   conn,
		raddr:  raddr,
		closed: make(chan struct{}),
	}

	go func() {
		// send heartbeat
		for {
			select {
			case <-c.closed:
				return
			case <-time.After(Heartbeat):
				c.write(nil)
			}
		}
	}()
	return c
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	if c.raddr == nil {
		return c.RemoteAddr()
	}
	return c.raddr
}

func (c *Conn) Read(b []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, syscall.EINVAL
	default:
		return c.r.read(b, c.raddr)
	}
}

func (c *Conn) write(b []byte) (int, error) {
	if c.raddr == nil {
		return c.conn.Write(b)
	}
	return c.conn.(net.PacketConn).WriteTo(b, c.raddr)
}

func (c *Conn) Write(b []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, syscall.EINVAL
	default:
		return c.write(b)
	}
}

func (c *Conn) Close() error {
	close(c.closed)
	return nil
}

// stub
func (c *Conn) SetDeadline(time.Time) error      { return nil }
func (c *Conn) SetReadDeadline(time.Time) error  { return nil }
func (c *Conn) SetWriteDeadline(time.Time) error { return nil }
