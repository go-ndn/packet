package packet

import (
	"io"
	"sync"
	"time"
)

type entry struct {
	time time.Time
	ch   chan byte
	mu   sync.Mutex
}

type reader struct {
	m  map[string]*entry
	mu sync.Mutex
}

func newReader() *reader {
	return &reader{m: make(map[string]*entry)}
}

func (r *reader) read(b []byte, saddr string) (n int, err error) {
	r.mu.Lock()
	ent, ok := r.m[saddr]
	r.mu.Unlock()
	if !ok {
		err = io.EOF
		return
	}
	for n = 0; n < len(b); n++ {
		select {
		case b[n] = <-ent.ch:
		case <-time.After(Dead):
			ent.mu.Lock()
			ok := time.Since(ent.time) < Dead
			ent.mu.Unlock()
			if n == 0 && !ok {
				r.mu.Lock()
				delete(r.m, saddr)
				r.mu.Unlock()

				err = io.EOF
			}
			return
		}
	}
	return
}

func (r *reader) write(b []byte, saddr string) (create bool) {
	r.mu.Lock()
	if _, ok := r.m[saddr]; !ok {
		r.m[saddr] = &entry{
			ch: make(chan byte, bufferSize),
		}
		create = true
	}
	ent := r.m[saddr]
	r.mu.Unlock()
	if cap(ent.ch)-len(ent.ch) < len(b) {
		return
	}
	ent.mu.Lock()
	ent.time = time.Now()
	ent.mu.Unlock()
	for _, bb := range b {
		ent.ch <- bb
	}
	return
}
