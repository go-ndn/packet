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

const (
	heartbeat  = 2 * time.Hour
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
				ticker.Stop()
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
