package atomicarena

import (
	"sync"
	"sync/atomic"
)

// AtomicRingArena is a fixed-size, bump-pointer ring allocator for T.
// Safe for concurrent use.
type AtomicArena[T any] struct {
	mu      sync.Mutex
	buf     []atomic.Pointer[T]
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
	buf := make([]atomic.Pointer[T], size)
	return &AtomicArena[T]{
		buf:  buf,
		size: uint64(size),
		mu:   sync.Mutex{},
	}
}

// Alloc atomically grabs the next slot, overwrites it with val,
// and returns a pointer to an independent copy of that slot.
func (a *AtomicArena[T]) Alloc(val T) *T {
	idx := atomic.AddUint64(&a.counter, 1) - 1
	slot := idx % a.size

	// copy value and store atomically in the buffer
	ptr := new(T)
	*ptr = val
	a.buf[slot].Store(ptr)

	// return independent copy
	out := new(T)
	*out = val
	return out
}

func (a *AtomicArena[T]) Reset() {
	atomic.StoreUint64(&a.counter, 0)
	for i := range a.buf {
		a.buf[i].Store(nil)
	}
}

func (a *AtomicArena[T]) PtrAt(i uint64) *T {
	return a.buf[i%a.size].Load()
}
