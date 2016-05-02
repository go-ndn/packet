package packet

import (
	"net"
	"strings"

	"github.com/go-ndn/tlv"
)

func dial(network, addr string) (net.Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	buf := newBuffer()
	go func() {
		b := make([]byte, tlv.MaxSize)
		for {
			n, err := c.Read(b)
			if err != nil {
				return
			}
			buf.Write(b[:n])
		}
	}()

	return newConn(buf, c, c, c.LocalAddr(), c.RemoteAddr()), nil
}

// Dial connects to the address on the named network.
//
// If the network is not packet-oriented, it calls net.Dial directly.
func Dial(network, address string) (net.Conn, error) {
	if strings.IndexByte(address, ':') == 0 {
		address = "localhost" + address
	}
	switch network {
	case "udp", "udp4", "udp6", "ip", "ip4", "ip6", "unixgram":
		return dial(network, address)
	default:
		return net.Dial(network, address)
	}
}
