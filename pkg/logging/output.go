package logging

import (
	"io"
	"os"
	"sync"
)

// processOut is the writer used by the process PlainHandler.
// taskui points it at Session.LogWriter while the progress TUI is up.
var processOut = &swapWriter{w: os.Stderr}

// ProcessWriter is the io.Writer for NewPlainHandler at process start.
func ProcessWriter() io.Writer { return processOut }

// SetProcessWriter redirects process log output. nil means os.Stderr.
func SetProcessWriter(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	processOut.Set(w)
}

type swapWriter struct {
	mu sync.Mutex
	w  io.Writer
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
