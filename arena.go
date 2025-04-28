package atomicarena

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// AtomicArena is a thread-safe lock-free arena allocator with a fixed maximum number of elements.
// It stores up to maxElems objects of type T. Alloc returns an error if the arena is full.
// T must be a type whose size is known at compile time.
type AtomicArena[T any] struct {
	buff     []atomic.Pointer[T] // pre-allocated slice of atomic pointers
	elemSize uintptr             // size of one element (not used directly but for consistency)
	maxElems uintptr             // maximum number of elements
	count    atomic.Uintptr      // number of elements allocated so far
}

// NewAtomicArena creates a new AtomicArena that can hold up to maxElems elements of type T.
// It pre-allocates the internal buffer accordingly.
func NewAtomicArena[T any](maxElems uintptr) *AtomicArena[T] {
	elemSize := unsafe.Sizeof(*new(T))
	return &AtomicArena[T]{
		buff:     make([]atomic.Pointer[T], maxElems),
		elemSize: elemSize,
		maxElems: maxElems,
	}
}

// Alloc atomically allocates obj within the arena. If the arena is full (maxElems reached), it returns an error.
func (a *AtomicArena[T]) Alloc(obj T) (*T, error) {
	// reserve slot
	for {
		old := a.count.Load()
		if old >= a.maxElems {
			return nil, fmt.Errorf("arena full: max elements %d exceeded", a.maxElems)
		}
		if a.count.CompareAndSwap(old, old+1) {
			// slot reserved at index old
			ptr := new(T)
			*ptr = obj

			// store pointer
			a.buff[old].Store(ptr)
			return ptr, nil
		}
	}
}

// AppendSlice atomically allocates a slice of objs within the arena. It returns a slice of pointers to the allocated objects.
// If there is not enough space to allocate all objs (maxElems exceeded), it returns an error without modifying the arena.
func (a *AtomicArena[T]) AppendSlice(objs []T) ([]*T, error) {
	n := uintptr(len(objs))
	// reserve a contiguous block of n slots
	for {
		old := a.count.Load()
		if old+n > a.maxElems {
			return nil, fmt.Errorf("arena full: cannot append slice of size %d, max elements %d exceeded", n, a.maxElems)
		}
		if a.count.CompareAndSwap(old, old+n) {
			// reserved block starting at index old
			ptrs := make([]*T, len(objs))
			for i, obj := range objs {
				ptr := new(T)
				*ptr = obj
				// store each pointer in the reserved slot
				a.buff[old+uintptr(i)].Store(ptr)
				ptrs[i] = ptr
			}
			return ptrs, nil
		}
	}
}

// Reset clears all allocations in the arena, allowing reuse.
func (a *AtomicArena[T]) Reset() {
	// clear stored pointers
	for i := uintptr(0); i < a.maxElems; i++ {
		a.buff[i].Store(nil)
	}
	// reset count
	a.count.Store(0)
}
