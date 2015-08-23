package packet

import "net"

func Dial(network, addr string) (net.Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	buf := newBuffer()
	// ensure ready to read
	buf.WriteTo("", nil)
	c := newConn(buf, conn, nil)
	go func() {
		b := make([]byte, packetSize)
		for {
			n, err := conn.Read(b)
			if err != nil {
				return
			}
			// use "" as address
			buf.WriteTo("", b[:n])
		}
	}()
	return c, nil
}
