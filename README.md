# stream-oriented packet connection

- [x] stream-oriented
- [x] heartbeat peer detection
- [x] fast shutdown
- [ ] reliable
- [ ] in-order

This provides minimal abstraction for NDN face. It uses 1-byte message to control the connection because the minimum TLV packet is 2-byte.

[![GoDoc](https://godoc.org/github.com/go-ndn/packet?status.svg)](https://godoc.org/github.com/go-ndn/packet)
