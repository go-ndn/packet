package packet

import (
	"io"
	"sync"
	"time"
)

type entry struct {
	b         chan byte
	keepAlive chan struct{}
	shutdown  chan struct{}
}

// buffer allows incoming messages to be distributed to many readers
type buffer struct {
	m map[string]entry
	sync.RWMutex
}

func newBuffer() *buffer {
	return &buffer{m: make(map[string]entry)}
}

func (buf *buffer) ReadFrom(saddr string, b []byte) (n int, err error) {
	defer func() {
		if err != nil {
			buf.Lock()
			delete(buf.m, saddr)
			buf.Unlock()
		}
	}()
	buf.RLock()
	ent, ok := buf.m[saddr]
	buf.RUnlock()
	if !ok {
		err = io.EOF
		return
	}
	// handle complex signals only on the first byte
	select {
	case b[0] = <-ent.b:
		// fill in as much as possible
		for n = 1; n < len(b); n++ {
			select {
			case b[n] = <-ent.b:
			default:
				return
			}
		}
	case <-ent.shutdown:
		err = io.EOF
		return
	case <-time.After(Heartbeat):
		select {
		case <-ent.keepAlive:
		default:
			err = io.EOF
		}
		return
	}
	return
}

func (buf *buffer) WriteTo(saddr string, b []byte) (create bool) {
	buf.Lock()
	ent, ok := buf.m[saddr]
	if !ok {
		create = true
		ent = entry{
			b:         make(chan byte, bufferSize),
			keepAlive: make(chan struct{}, 1),
			shutdown:  make(chan struct{}, 1),
		}
		buf.m[saddr] = ent
	}
	buf.Unlock()

	switch len(b) {
	case 0:
	case 1:
		switch b[0] {
		case keepAlive[0]:
			select {
			case ent.keepAlive <- struct{}{}:
			default:
			}
		case shutdown[0]:
			select {
			case ent.shutdown <- struct{}{}:
			default:
			}
		}
	default:
		if cap(ent.b)-len(ent.b) < len(b) {
			return
		}
		for _, bb := range b {
			ent.b <- bb
		}
	}
	return
}
