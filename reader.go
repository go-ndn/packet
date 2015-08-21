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

type reader struct {
	m map[string]entry
	sync.RWMutex
}

func newReader() *reader {
	return &reader{m: make(map[string]entry)}
}

func (r *reader) read(b []byte, saddr string) (n int, err error) {
	defer func() {
		if err != nil {
			r.Lock()
			delete(r.m, saddr)
			r.Unlock()
		}
	}()
	r.RLock()
	ent, ok := r.m[saddr]
	r.RUnlock()
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

func (r *reader) write(b []byte, saddr string) (create bool) {
	r.RLock()
	ent, ok := r.m[saddr]
	r.RUnlock()

	if !ok {
		create = true
		ent = entry{
			b:         make(chan byte, bufferSize),
			keepAlive: make(chan struct{}, 1),
			shutdown:  make(chan struct{}, 1),
		}
		r.Lock()
		r.m[saddr] = ent
		r.Unlock()
	}
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
