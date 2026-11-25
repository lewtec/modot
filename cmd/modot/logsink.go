package main

import (
	"io"
	"sync"
)

// swapWriter is the process log destination. The slog handler holds this
// for the life of the process; Set points it at Session.LogWriter
// while progress.Run is in flight.
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
