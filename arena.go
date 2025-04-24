package atomicarena

import (
	"errors"
	"sync"
	"sync/atomic"
)

// AtomicArena is a fixed-size, bump-pointer allocator for T.
// Allocations are lock-free; Reset should not be called concurrently with Alloc.
type AtomicArena[T any] struct {
	buf     []atomic.Pointer[T]
	size    uint64
	counter uint64
	mu      sync.Mutex
	_       [8]uint64 // padding for alignment
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
// Uses atomic operations for lock-free allocation.
func (a *AtomicArena[T]) Alloc(val T) (*T, error) {
	idx := atomic.AddUint64(&a.counter, 1) - 1
	if idx >= a.size {
		// rollback counter to avoid overflow
		atomic.AddUint64(&a.counter, ^uint64(0))
		return nil, errors.New("arena is full")
	}
	// allocate pointer and set value
	ptr := new(T)
	*ptr = val
	// ensure data written before publishing pointer
	a.buf[idx].Store(ptr)
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

// Reset clears all slots up to the current counter and resets it to zero.
// Not safe to call concurrently with Alloc.
func (a *AtomicArena[T]) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// clear only used slots
	n := atomic.LoadUint64(&a.counter)
	if n > a.size {
		n = a.size
	}
	for i := uint64(0); i < n; i++ {
		a.buf[i].Store(nil)
	}
	// reset counter
	atomic.StoreUint64(&a.counter, 0)
}

// PtrAt returns the pointer at index i modulo arena size,
// or nil if the slot hasn't been allocated since last Reset.
func (a *AtomicArena[T]) PtrAt(i uint64) *T {
	if a.size == 0 {
		return nil
	}
	idx := i % a.size
	return a.buf[idx].Load()
}

// MakeSlice returns a slice of all allocated pointers in order.
func (a *AtomicArena[T]) MakeSlice() ([]*T, error) {
	n := atomic.LoadUint64(&a.counter)
	if n > a.size {
		n = a.size
	}
	result := make([]*T, n)
	for i := uint64(0); i < n; i++ {
		result[i] = a.buf[i].Load()
	}
	return result, nil
}

// AppendSlice appends each value in vals into the existing slice dest
// by allocating them in the arena, returning the updated slice.
// If the arena becomes full before all values are stored, it returns an error.
func (a *AtomicArena[T]) AppendSlice(dest []*T, vals []T) ([]*T, error) {
	for _, val := range vals {
		ptr, err := a.Alloc(val)
		if err != nil {
			return dest, err
		}
		dest = append(dest, ptr)
	}
	return dest, nil
}
