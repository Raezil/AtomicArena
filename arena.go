package atomicarena

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// AtomicArena is a lock-free arena allocator with a fixed capacity in bytes.
// When the total size of allocations exceeds the specified capacity, Alloc returns an error.
// T must be a type whose size is known at compile time.

// AtomicArena holds the buffer of atomic pointers, the current offset, and the overall capacity.
type AtomicArena[T any] struct {
	buff     []atomic.Pointer[T]
	offset   uintptr
	capacity uintptr
}

// NewAtomicArena creates a new AtomicArena with the given capacity in bytes.
// Internally, it pre-allocates the buffer slice with capacity equal to
// capacity/sizeof(T) pointers, to avoid reallocations until the arena is full.
func NewAtomicArena[T any](capacity uintptr) *AtomicArena[T] {
	elemSize := unsafe.Sizeof(*new(T))
	maxElems := capacity / elemSize
	return &AtomicArena[T]{
		buff:     make([]atomic.Pointer[T], 0, maxElems),
		capacity: capacity,
	}
}

// Alloc allocates obj within the arena. If the arena is full (i.e. adding this allocation
// would exceed the capacity), it returns an error rather than growing the arena.
func (mem *AtomicArena[T]) Alloc(obj T) (*T, error) {
	sz := unsafe.Sizeof(obj)
	if mem.offset+sz > mem.capacity {
		return nil, fmt.Errorf("arena full: capacity %d bytes exceeded by request of %d bytes", mem.capacity, sz)
	}

	ptr := new(T)
	*ptr = obj

	var atomicPtr atomic.Pointer[T]
	atomicPtr.Store(ptr)

	mem.buff = append(mem.buff, atomicPtr)
	mem.offset += sz

	return ptr, nil
}

// Reset clears all allocations in the arena without freeing the underlying memory.
// It zeroes each atomic pointer and resets the offset back to zero.
func (mem *AtomicArena[T]) Reset() {
	for i := range mem.buff {
		mem.buff[i].Store(nil)
	}
	mem.offset = 0
}
