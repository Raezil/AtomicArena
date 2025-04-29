package atomicarena

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// AtomicArena is a thread-safe lock-free arena allocator with a fixed maximum number of elements.
// It stores up to maxElems objects of type T. Alloc returns an error if the arena is full.
// This optimized version avoids per-object heap allocations by pre-allocating a contiguous buffer of Ts.
// It also uses a single atomic.Add for bulk slice allocations.
type AtomicArena[T any] struct {
	raw      []T                 // contiguous storage for objects
	ptrs     []atomic.Pointer[T] // atomic pointers into raw, for tests and visibility
	maxElems uintptr             // maximum number of elements
	count    atomic.Uintptr      // number of elements allocated so far
}

// NewAtomicArena creates a new AtomicArena that can hold up to maxElems elements of type T.
// It pre-allocates both the raw buffer and the pointer slice.
func NewAtomicArena[T any](maxElems uintptr) *AtomicArena[T] {
	raw := make([]T, maxElems)
	ptrs := make([]atomic.Pointer[T], maxElems)
	return &AtomicArena[T]{
		raw:      raw,
		ptrs:     ptrs,
		maxElems: maxElems,
	}
}

// Alloc atomically reserves one slot and stores obj in the pre-allocated buffer.
// Returns a pointer to the stored object, or error if full.
func (a *AtomicArena[T]) Alloc(obj T) (*T, error) {
	idx := a.count.Add(1) - 1
	if idx >= a.maxElems {
		// revert count
		a.count.Add(^uintptr(0))
		return nil, fmt.Errorf("arena full: max elements %d exceeded", a.maxElems)
	}
	// place object in raw buffer and publish pointer
	a.raw[idx] = obj
	a.ptrs[idx].Store(&a.raw[idx])
	return &a.raw[idx], nil
}

// AppendSlice reserves n slots in one atomic.Add and fills them with objs.
// Returns slice of pointers to the stored objects, or error if not enough space.
func (a *AtomicArena[T]) AppendSlice(objs []T) ([]*T, error) {
	n := uintptr(len(objs))
	if n == 0 {
		return nil, nil
	}
	// Bulk reserve
	start := a.count.Add(n) - n
	if start+n > a.maxElems {
		// revert reservation
		a.count.Add(^uintptr(n - 1))
		return nil, fmt.Errorf("arena full: cannot append slice of size %d, max elements %d exceeded", n, a.maxElems)
	}
	// fill and publish
	ptrs := make([]*T, len(objs))
	for i, obj := range objs {
		a.raw[start+uintptr(i)] = obj
		a.ptrs[start+uintptr(i)].Store(&a.raw[start+uintptr(i)])
		ptrs[i] = &a.raw[start+uintptr(i)]
	}
	return ptrs, nil
}

//go:linkname memclrNoHeapPointers runtime.memclrNoHeapPointers
//go:nosplit
func memclrNoHeapPointers(ptr unsafe.Pointer, n uintptr)

// Reset clears all published pointers, allowing reuse of the arena.
// It zeroes the ptrs slice via memclrNoHeapPointers and resets the allocation count.
func (a *AtomicArena[T]) Reset() {
	old := a.count.Load()
	if old > 0 {
		// zero out published pointers
		ptr := unsafe.Pointer(&a.ptrs[0])
		sz := unsafe.Sizeof(a.ptrs[0])
		memclrNoHeapPointers(ptr, old*sz)
	}
	// allow reuse
	a.count.Store(0)
}
