package proxy

import "sync"

type sharedVariables struct {
	mu  sync.Mutex
	cnt uint64
}

func (s *sharedVariables) counter() uint64 {
	var c uint64
	s.mu.Lock()
	c = s.cnt
	s.mu.Unlock()
	return c
}

func (s *sharedVariables) counterInc() {
	s.mu.Lock()
	s.cnt += 1
	s.mu.Unlock()
}
