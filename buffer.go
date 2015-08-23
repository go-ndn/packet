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
	timer := time.NewTimer(Heartbeat)
	for n = 0; n < len(b); n++ {
		select {
		case b[n] = <-ent.b:
		case <-ent.shutdown:
			err = io.EOF
			return
		case <-timer.C:
			select {
			case <-ent.keepAlive:
			default:
				err = io.EOF
			}
			return
		}
		timer.Reset(Heartbeat)
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
		case keepAlive:
			select {
			case ent.keepAlive <- struct{}{}:
			default:
			}
		case shutdown:
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
