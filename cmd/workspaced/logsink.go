package main

import (
	"io"
	"sync"
)

// swapWriter is the process log destination. PlainHandler holds this
// for the life of the process; Set switches stderr vs the progress
// session LineWriter after the TUI is up. The TUI writer must stay
// installed until progress.Run returns (Wait), not only until app.Run.
type swapWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSwapWriter(w io.Writer) *swapWriter {
	return &swapWriter{w: w}
}

func (s *swapWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	w := s.w
	s.mu.Unlock()
	if w == nil {
		return len(p), nil
	}
	return w.Write(p)
}

func (s *swapWriter) Set(w io.Writer) {
	s.mu.Lock()
	s.w = w
	s.mu.Unlock()
}
