package atomicarena

import (
	"errors"
	"sync"
	"sync/atomic"
)

// AtomicArena is a fixed-size, bump-pointer allocator for T.
// Safe for concurrent use.
type AtomicArena[T any] struct {
	mu      sync.Mutex
	buf     []atomic.Pointer[T]
	size    uint64
	counter uint64
	_       [8]uint64 // padding
}

// NewAtomicArena creates an arena of exactly `size` slots.
// Returns an error if size <= 0.
func NewAtomicArena[T any](size int) (*AtomicArena[T], error) {
	if size <= 0 {
		return nil, errors.New("size must be > 0")
	}
	buf := make([]atomic.Pointer[T], size)
	return &AtomicArena[T]{
		buf:  buf,
		size: uint64(size),
	}, nil
}

// Alloc stores val in the next free slot, or returns an error if full.
func (a *AtomicArena[T]) Alloc(val T) (*T, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.counter >= a.size {
		return nil, errors.New("arena is full")
	}

	slot := a.counter
	a.counter++

	ptr := new(T)
	*ptr = val
	a.buf[slot].Store(ptr)
	return ptr, nil
}

// Reset clears all slots and resets the allocation counter.
func (a *AtomicArena[T]) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.counter = 0
	for i := range a.buf {
		a.buf[i].Store(nil)
	}
}

// PtrAt returns the pointer at index modulo arena size.
func (a *AtomicArena[T]) PtrAt(i uint64) *T {
	if a.size == 0 {
		return nil
	}
	return a.buf[i%a.size].Load()
}
