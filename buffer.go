package packet

import (
	"io"
	"time"
)

// buffer allows incoming messages to be distributed to many readers
type buffer struct {
	b         chan byte
	keepAlive chan struct{}
	shutdown  chan struct{}
}

const (
	bufferSize    = 131072
	heartbeatIntv = 4 * time.Second
)

func newBuffer() io.ReadWriter {
	return &buffer{
		b:         make(chan byte, bufferSize),
		keepAlive: make(chan struct{}, 10),
		shutdown:  make(chan struct{}, 1),
	}
}

func (buf *buffer) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		err = io.ErrShortBuffer
		return
	}
	// handle complex signals only on the first byte
	select {
	case b[0] = <-buf.b:
		// fill in as much as possible
		for n = 1; n < len(b); n++ {
			select {
			case b[n] = <-buf.b:
			default:
				return
			}
		}
	case <-buf.shutdown:
		err = io.EOF
	case <-time.After(heartbeatIntv):
		select {
		case <-buf.keepAlive:
		default:
			err = io.EOF
		}
	}
	return
}

func (buf *buffer) Write(b []byte) (n int, err error) {
	switch len(b) {
	case 0:
	case 1:
		switch b[0] {
		case keepAlive[0]:
			select {
			case buf.keepAlive <- struct{}{}:
			default:
			}
		case shutdown[0]:
			select {
			case buf.shutdown <- struct{}{}:
			default:
			}
		}
	default:
		if cap(buf.b)-len(buf.b) < len(b) {
			return
		}
		for _, bb := range b {
			buf.b <- bb
		}
	}
	return
}
