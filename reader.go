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
	m  map[string]entry
	mu sync.RWMutex
}

func newReader() *reader {
	return &reader{m: make(map[string]entry)}
}

func (r *reader) read(b []byte, saddr string) (n int, err error) {
	defer func() {
		if err == io.EOF {
			r.mu.Lock()
			delete(r.m, saddr)
			r.mu.Unlock()
		}
	}()
	r.mu.RLock()
	ent, ok := r.m[saddr]
	r.mu.RUnlock()
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
	r.mu.RLock()
	ent, ok := r.m[saddr]
	r.mu.RUnlock()

	if !ok {
		create = true
		ent = entry{
			b:         make(chan byte, bufferSize),
			keepAlive: make(chan struct{}, 1),
			shutdown:  make(chan struct{}, 1),
		}
		r.mu.Lock()
		r.m[saddr] = ent
		r.mu.Unlock()
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
