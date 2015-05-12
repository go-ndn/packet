package packet

import (
	"io"
	"sync"
	"time"
)

type entry struct {
	b   chan byte
	oob chan struct{}
}

type reader struct {
	m  map[string]entry
	mu sync.RWMutex
}

func newReader() *reader {
	return &reader{m: make(map[string]entry)}
}

func (r *reader) read(b []byte, saddr string) (n int, err error) {
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
		case <-timer.C:
			select {
			case <-ent.oob:
			default:
				r.mu.Lock()
				delete(r.m, saddr)
				r.mu.Unlock()

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
			b:   make(chan byte, bufferSize),
			oob: make(chan struct{}, 1),
		}
		r.mu.Lock()
		r.m[saddr] = ent
		r.mu.Unlock()
	}
	if cap(ent.b)-len(ent.b) < len(b) {
		return
	}
	for _, bb := range b {
		ent.b <- bb
	}
	select {
	case ent.oob <- struct{}{}:
	default:
	}
	return
}
