package packet

import (
	"io"
	"net"
	"time"
)

func Dial(network, addr string) (net.Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	d := &dialer{
		conn: conn,
		entry: entry{
			ch: make(chan byte, bufferSize),
		},
	}
	c := newConn(d, conn, nil)
	go func() {
		b := make([]byte, packetSize)
		for {
			select {
			case <-c.closed:
				return
			default:
				conn.SetReadDeadline(time.Now().Add(Dead))
				n, err := conn.Read(b)
				if err != nil {
					continue
				}
				if cap(d.ch)-len(d.ch) < n {
					continue
				}
				d.mu.Lock()
				d.time = time.Now()
				d.mu.Unlock()
				for _, bb := range b[:n] {
					d.ch <- bb
				}
			}
		}
	}()
	return c, nil
}

type dialer struct {
	entry
	conn net.Conn
}

func (d *dialer) read(b []byte, _ net.Addr) (n int, err error) {
	for n = 0; n < len(b); n++ {
		select {
		case b[n] = <-d.ch:
		case <-time.After(Dead):
			d.mu.Lock()
			ok := time.Now().Sub(d.time) < Dead
			d.mu.Unlock()
			if n == 0 && !ok {
				err = io.EOF
			}
			return
		}
	}
	return
}
