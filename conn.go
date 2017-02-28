// Package packet implements stream-oriented packet connection.
//
// The implementation splits packet-oriented connection like UDP and IP into different streams
// with remote address.
// In addition, heatbeat is sent periodically to detect whether remote is reachable.
//
// Although timeout is not supported, every network supported by go including net.PacketConn
// can be converted to net.Conn with packet.Dial and packet.Listen. Network reliability remains
// the same; for example, UDP is still unreliable.
//
// One-byte packet is reserved for managing streams.
package packet

import (
	"io"
	"net"
	"syscall"
	"time"
)

var (
	keepAlive = []byte{0x01}
	shutdown  = []byte{0x02}
)

type conn struct {
	io.Reader
	io.Writer
	io.Closer
	laddr, raddr net.Addr
	closed       chan struct{}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) {
	return f(b)
}

type closerFunc func() error

func (f closerFunc) Close() error {
	return f()
}

func newConn(r io.Reader, w io.Writer, closer io.Closer, laddr, raddr net.Addr) *conn {
	c := &conn{
		Reader: r,
		Writer: w,
		Closer: closer,
		laddr:  laddr,
		raddr:  raddr,
		closed: make(chan struct{}),
	}

	go func() {
		// notify the other end even if no message is given
		c.Write(keepAlive)
		// send heartbeat
		ticker := time.NewTicker(heartbeatIntv)
		defer ticker.Stop()
		for {
			select {
			case <-c.closed:
				return
			case <-ticker.C:
				c.Write(keepAlive)
			}
		}
	}()
	return c
}

func (c *conn) LocalAddr() net.Addr {
	return c.laddr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.raddr
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

func (c *conn) Write(b []byte) (int, error) {
	select {
	case <-c.closed:
		// already closed
		return 0, syscall.EINVAL
	default:
		return c.Writer.Write(b)
	}
}

func (c *conn) Close() error {
	// write shutdown signal
	c.Write(shutdown)
	close(c.closed)
	return c.Closer.Close()
}

func (c *conn) SetDeadline(time.Time) error      { return nil }
func (c *conn) SetReadDeadline(time.Time) error  { return nil }
func (c *conn) SetWriteDeadline(time.Time) error { return nil }
