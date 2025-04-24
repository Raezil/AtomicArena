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

// AllocSlice stores each value in vals into the arena using Alloc,
// returning a slice of pointers to the stored values. If the arena
// becomes full before all values are stored, it returns an error.
func (a *AtomicArena[T]) AllocSlice(vals []T) ([]*T, error) {
	pointers := make([]*T, 0, len(vals))
	for _, val := range vals {
		ptr, err := a.Alloc(val)
		if err != nil {
			return nil, err
		}
		pointers = append(pointers, ptr)
	}
	return pointers, nil
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

// MakeSlice ensures the arena is full by using AllocSlice to fill
// any remaining slots with zero values, then returns a slice of all pointers in order.
func (a *AtomicArena[T]) MakeSlice() ([]*T, error) {
	// Determine how many slots are unfilled
	a.mu.Lock()
	filled := a.counter
	total := a.size
	a.mu.Unlock()

	missing := int(total - filled)
	if missing > 0 {
		// Prepare a zero-valued slice to fill missing slots
		zeros := make([]T, missing)
		if _, err := a.AllocSlice(zeros); err != nil {
			return nil, err
		}
	}

	// Build the full slice
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make([]*T, total)
	for i := uint64(0); i < total; i++ {
		result[i] = a.buf[i].Load()
	}
	return result, nil
}
