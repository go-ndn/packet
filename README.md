# stream-oriented packet connection

- [x] stream-oriented
- [x] heartbeat
- [x] fast shutdown
- [ ] reliable
- [ ] in-order
- [ ] flow control

It uses 1-byte message to control the connection because the minimum TLV packet is 2-byte.

[![GoDoc](https://godoc.org/github.com/go-ndn/packet?status.svg)](https://godoc.org/github.com/go-ndn/packet)
