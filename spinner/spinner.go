package spinner

import (
	"fmt"
	"sync"
	"time"

	"github.com/titpetric/atkins-ci/colors"
)

// Spinner displays an animated three-dot spinner with one green dot rotating
type Spinner struct {
	position int
	mu       sync.Mutex
	done     chan bool
	running  bool
}

// New creates a new spinner
func New() *Spinner {
	return &Spinner{
		position: 0,
		done:     make(chan bool),
		running:  false,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.position = (s.position + 1) % 3
				s.mu.Unlock()
			}
		}
	}()
}

// Stop halts the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.done <- true
}

// String returns the current spinner state
func (s *Spinner) String() string {
	s.mu.Lock()
	pos := s.position
	s.mu.Unlock()

	dots := [3]string{"○", "○", "○"}
	dots[pos] = "●"

	return fmt.Sprintf("%s%s%s",
		colors.BrightGreen(dots[0]),
		colors.BrightCyan(dots[1]),
		colors.BrightCyan(dots[2]),
	)
}
