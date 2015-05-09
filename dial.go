package packet

import (
	"net"
	"time"
)

func Dial(network, addr string) (net.Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	r := newReader()
	// ensure entry exist
	r.write(nil, "")
	c := newConn(r, conn, nil)
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
				r.write(b[:n], "")
			}
		}
	}()
	return c, nil
}
