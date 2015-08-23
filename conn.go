package packet

import (
	"net"
	"syscall"
	"time"
)

const (
	Heartbeat  = 10 * time.Second
	packetSize = 8192
	bufferSize = 131072
)

var (
	keepAlive = []byte{0x01}
	shutdown  = []byte{0x02}
)

type conn struct {
	// read
	*buffer
	// write
	net.Conn
	raddr net.Addr
	// close
	closed chan struct{}
}

func newConn(buf *buffer, netConn net.Conn, raddr net.Addr) net.Conn {
	c := &conn{
		buffer: buf,
		Conn:   netConn,
		raddr:  raddr,
		closed: make(chan struct{}),
	}

	go func() {
		// notify the other end even if no message is given
		c.write(keepAlive)
		// send heartbeat
		ticker := time.NewTicker(Heartbeat)
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
	if c.raddr == nil {
		// dialer
		return c.Conn.RemoteAddr()
	}
	// listener
	return c.raddr
}

func (c *conn) Read(b []byte) (int, error) {
	select {
	case <-c.closed:
		// already closed
		return 0, syscall.EINVAL
	default:
		// read from buffer with address
		var saddr string
		if c.raddr != nil {
			saddr = c.raddr.String()
		}
		return c.ReadFrom(saddr, b)
	}
}

func (c *conn) write(b []byte) (int, error) {
	if c.raddr == nil {
		// dialer
		return c.Conn.Write(b)
	}
	// listener
	return c.Conn.(net.PacketConn).WriteTo(b, c.raddr)
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
	if c.raddr == nil {
		// dialer
		return c.Conn.Close()
	}
	return nil
}

func (c *conn) SetDeadline(time.Time) error      { return nil }
func (c *conn) SetReadDeadline(time.Time) error  { return nil }
func (c *conn) SetWriteDeadline(time.Time) error { return nil }
