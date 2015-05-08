package packet

import (
	"net"
	"syscall"
	"time"
)

const (
	Heartbeat  = 10 * time.Second
	Dead       = 2 * Heartbeat
	packetSize = 8192
	bufferSize = 131072
)

type Conn struct {
	r      reader
	conn   net.Conn
	raddr  net.Addr
	closed chan struct{}
}

type reader interface {
	read([]byte, net.Addr) (int, error)
}

func newConn(r reader, conn net.Conn, raddr net.Addr) *Conn {
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
