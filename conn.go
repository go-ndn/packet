package packet

import (
	"io"
	"net"
	"syscall"
	"time"
)

const (
	heartbeat  = 2 * time.Hour
	packetSize = 8192
	bufferSize = 131072
)

var (
	keepAlive = []byte{0x01}
	shutdown  = []byte{0x02}
)

type conn struct {
	// read
	io.Reader
	// write
	net.Conn
	net.Addr
	// close
	closed chan struct{}
}

func newConn(r io.Reader, netConn net.Conn, raddr net.Addr) *conn {
	c := &conn{
		Reader: r,
		Conn:   netConn,
		Addr:   raddr,
		closed: make(chan struct{}),
	}

	go func() {
		// notify the other end even if no message is given
		c.write(keepAlive)
		// send heartbeat
		ticker := time.NewTicker(heartbeat)
		for {
			select {
			case <-c.closed:
				return
			case <-ticker.C:
				c.write(keepAlive)
			}
		}
	}()
	return c
}

func (c *conn) RemoteAddr() net.Addr {
	if c.Addr == nil {
		// dialer
		return c.Conn.RemoteAddr()
	}
	// listener
	return c.Addr
}

func (c *conn) Read(b []byte) (int, error) {
	select {
	case <-c.closed:
		// already closed
		return 0, syscall.EINVAL
	default:
		return c.Reader.Read(b)
	}
}

func (c *conn) write(b []byte) (int, error) {
	if c.Addr == nil {
		// dialer
		return c.Conn.Write(b)
	}
	// listener
	return c.Conn.(net.PacketConn).WriteTo(b, c.Addr)
}

func (c *conn) Write(b []byte) (int, error) {
	select {
	case <-c.closed:
		// already closed
		return 0, syscall.EINVAL
	default:
		return c.write(b)
	}
}

func (c *conn) Close() error {
	// write shutdown signal
	c.write(shutdown)
	close(c.closed)
	if c.Addr == nil {
		// dialer
		return c.Conn.Close()
	}
	return nil
}

func (c *conn) SetDeadline(time.Time) error      { return nil }
func (c *conn) SetReadDeadline(time.Time) error  { return nil }
func (c *conn) SetWriteDeadline(time.Time) error { return nil }
