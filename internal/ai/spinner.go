package ai

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const spinnerInterval = 100 * time.Millisecond

var spinnerFrames = []string{"|", "/", "-", "\\"}

// Spinner displays an animated thinking indicator on w. Call Stop to halt it.
// The spinner only runs when isTerminal is true; otherwise it is a no-op.
// Stop may be called multiple times safely.
type Spinner struct {
	w          io.Writer
	stop       chan struct{}
	done       chan struct{}
	isTerminal bool
	once       sync.Once
}

// NewSpinner creates a Spinner that writes to w. Start it with Run.
func NewSpinner(w io.Writer, isTerminal bool) *Spinner {
	return &Spinner{
		w:          w,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		isTerminal: isTerminal,
	}
}

// Run starts the spinner in the background. It returns immediately.
func (s *Spinner) Run() {
	if !s.isTerminal {
		close(s.done)
		return
	}
	go func() {
		defer close(s.done)
		i := 0
		for {
			fmt.Fprintf(s.w, "\r\033[K%s Thinking...", spinnerFrames[i%len(spinnerFrames)])
			i++
			select {
			case <-s.stop:
				// Clear the spinner line so streamed text starts clean.
				fmt.Fprint(s.w, "\r\033[K")
				return
			case <-time.After(spinnerInterval):
			}
		}
	}()
}

// Stop halts the spinner and waits for the goroutine to finish.
// Safe to call multiple times.
func (s *Spinner) Stop() {
	s.once.Do(func() {
		if s.isTerminal {
			close(s.stop)
		}
	})
	<-s.done
}
