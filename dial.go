package packet

import "net"

func dial(network, addr string) (net.Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	buf := newBuffer()
	go func() {
		b := make([]byte, packetSize)
		for {
			n, err := conn.Read(b)
			if err != nil {
				return
			}
			buf.Write(b[:n])
		}
	}()
	return newConn(buf, conn, nil), nil
}

func Dial(network, address string) (net.Conn, error) {
	switch network {
	case "udp", "udp4", "udp6", "ip", "ip4", "ip6", "unixgram":
		return dial(network, address)
	default:
		return net.Dial(network, address)
	}
}
