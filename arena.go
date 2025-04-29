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
	elemSize uintptr             // size of one element (for consistency)
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
// We use atomic.Add to reserve a slot and ensure proper memory ordering, and revert the count if out of bounds.
func (a *AtomicArena[T]) Alloc(obj T) (*T, error) {
	// reserve slot by incrementing count atomically
	idx := a.count.Add(1) - 1
	if idx >= a.maxElems {
		// revert count since we've gone over capacity
		a.count.Add(^uintptr(0))
		return nil, fmt.Errorf("arena full: max elements %d exceeded", a.maxElems)
	}

	// allocate object and store pointer
	ptr := new(T)
	*ptr = obj
	a.buff[idx].Store(ptr)
	return ptr, nil
}

// AppendSlice atomically allocates a slice of objs within the arena. It returns a slice of pointers to the allocated objects.
// If there is not enough space to allocate all objs (maxElems exceeded), it returns an error without modifying the arena.
// This version properly handles allocation after Reset() by always creating fresh objects.
func (a *AtomicArena[T]) AppendSlice(objs []T) ([]*T, error) {
	n := uintptr(len(objs))
	if n == 0 {
		return nil, nil
	}

	// Reserve the necessary space atomically
	for {
		old := a.count.Load()
		newCount := old + n

		if newCount > a.maxElems {
			return nil, fmt.Errorf("arena full: cannot append slice of size %d, max elements %d exceeded", n, a.maxElems)
		}

		if a.count.CompareAndSwap(old, newCount) {
			// Allocation successful, now populate the arena
			ptrs := make([]*T, len(objs))
			for i, obj := range objs {
				// Always create new objects for each allocation
				ptr := new(T)
				*ptr = obj
				a.buff[old+uintptr(i)].Store(ptr)
				ptrs[i] = ptr
			}
			return ptrs, nil
		}
		// CAS failed, another thread changed the count, retry
	}
}

//go:linkname memclrNoHeapPointers runtime.memclrNoHeapPointers
//go:nosplit
func memclrNoHeapPointers(ptr unsafe.Pointer, n uintptr)

// Reset clears all allocations in the arena, allowing reuse.
// It resets the allocation count first to prevent readers from accessing stale pointers.
// Reset clears all allocations in the arena, allowing reuse.
// It resets the allocation count and zeroes out all allocated memory.
// Reset clears the arena by zeroing memory without heap pointers, then resets the counter.
// It uses memclrNoHeapPointers for efficient bulk zeroing.
func (a *AtomicArena[T]) Reset() {
	oldCount := a.count.Load()
	if oldCount > 0 {
		// Calculate memory region for current values
		ptr := unsafe.Pointer(&a.buff[0])
		size := unsafe.Sizeof(a.buff[0])
		memclrNoHeapPointers(ptr, oldCount*size)
	}
	// Reset count to allow reuse
	a.count.Store(0)
}
