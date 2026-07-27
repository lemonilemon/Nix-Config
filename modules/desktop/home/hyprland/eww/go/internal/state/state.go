package state

import (
	"sync"

	"ewwbar/internal/pyjson"
)

// Store is BarState: the live state plus the signal the emit loop waits on.
//
// The Python holds a dict and a threading.Event and mutates by key. Go gets a
// struct and a mutator callback instead, so a typo in a field name is a compile
// error rather than a new key silently appearing in the snapshot -- which is
// the one thing a dict-based model cannot catch and eww would render as a
// missing widget.
type Store struct {
	mu      sync.Mutex
	bar     Bar
	changed chan struct{}
}

// New returns a Store holding the same initial value as BarState().
func New() *Store {
	return &Store{
		bar: Default(),
		// Buffered, size 1, and non-blocking sends: this is a coalescing
		// signal, not a queue. threading.Event has the same shape -- ten
		// updates between two emits produce one wakeup, and the emit reads
		// whatever the state is at that moment.
		changed: make(chan struct{}, 1),
	}
}

// Changed fires when the state has been modified since the last read.
func (s *Store) Changed() <-chan struct{} { return s.changed }

// Update applies mutate under the lock and signals only if something actually
// changed, matching BarState.update's compare-then-set.
//
// The equality test is on the encoded snapshot rather than the struct, because
// Bar contains slices and Go cannot compare those with ==. That costs one
// encode per update, against a state this size -- measured at ~4 KB -- which is
// cheaper than the Python's per-key dict comparison plus its own encode.
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

// Snapshot is BarState.snapshot: the exact bytes eww reads on stdout.
//
// ensure_ascii=False, matching state.py:72. The control socket uses the other
// setting; see pyjson.Encode.
func (s *Store) Snapshot() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return pyjson.Encode(s.bar, false)
}

// Get returns a copy of the current state.
func (s *Store) Get() Bar {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bar
}
