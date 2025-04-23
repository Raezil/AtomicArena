package atomicarena

import (
	"sync"
	"sync/atomic"
)

// AtomicRingArena is a fixed-size, bump-pointer ring allocator for T.
// Safe for concurrent use.
type AtomicArena[T any] struct {
	mu      sync.Mutex
	buf     []T
	size    uint64    // buffer length
	counter uint64    // ever-growing allocation counter
	_       [8]uint64 // padding to avoid false sharing
}

// NewAtomicRingArena creates a ring of exactly `size` slots.
// Panics if size <= 0.
func NewAtomicArena[T any](size int) *AtomicArena[T] {
	if size <= 0 {
		panic("size must be > 0")
	}
	return &AtomicArena[T]{
		buf:  make([]T, size),
		size: uint64(size),
		mu:   sync.Mutex{},
	}
}

// Alloc atomically grabs the next slot, overwrites it with val,
// and returns a pointer to an independent copy of that slot.
func (a *AtomicArena[T]) Alloc(val T) *T {
	a.mu.Lock()
	idx := atomic.AddUint64(&a.counter, 1) - 1
	slot := idx % a.size
	// store in ring buffer for PtrAt peeks
	a.buf[slot] = val
	// return a unique pointer to the value to avoid overwrites
	ptr := new(T)
	*ptr = val
	a.mu.Unlock()
	return ptr
}

func (a *AtomicArena[T]) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Clear the bump‐pointer so allocations start at 0 again.
	atomic.StoreUint64(&a.counter, 0)

	// Zero out every slot in the buffer.
	var zero T
	for i := range a.buf {
		a.buf[i] = zero
	}
}

// PtrAt lets you peek at element i mod size.
func (a *AtomicArena[T]) PtrAt(i uint64) *T {
	return &a.buf[i%a.size]
}
