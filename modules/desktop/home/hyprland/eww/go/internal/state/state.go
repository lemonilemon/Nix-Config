package state

import (
	"sync"

	"ewwbar/internal/pyjson"
)

// Store holds the live bar state and the signal the emit loop waits on.
type Store struct {
	mu      sync.Mutex
	bar     Bar
	changed chan struct{}
}

func New() *Store {
	return &Store{
		bar: Default(),
		// Buffered size 1 with non-blocking sends: a coalescing signal, not a queue.
		changed: make(chan struct{}, 1),
	}
}

func (s *Store) Changed() <-chan struct{} { return s.changed }

// Update applies mutate under the lock and signals only on a real change. The
// equality test is on the encoded snapshot because Bar holds slices.
func (s *Store) Update(mutate func(*Bar)) {
	s.mu.Lock()
	before, err := pyjson.Encode(s.bar, false)
	if err != nil {
		s.mu.Unlock()
		return
	}
	mutate(&s.bar)
	after, encodeErr := pyjson.Encode(s.bar, false)
	s.mu.Unlock()

	if encodeErr != nil || before == after {
		return
	}
	select {
	case s.changed <- struct{}{}:
	default: // already signalled and not yet consumed
	}
}

// Snapshot is the exact bytes eww reads on stdout, ensure_ascii=False. The
// control socket uses the other setting; see pyjson.Encode.
func (s *Store) Snapshot() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return pyjson.Encode(s.bar, false)
}

func (s *Store) Get() Bar {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bar
}
