package atomicarena

import (
	"errors"
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

var ErrArenaFull = errors.New("atomicarena: arena full")

// Reserve atomically reserves n slots and returns a slice view of length n.
// Caller may write directly into the returned slice. No copying of data is performed.
func (a *AtomicArena[T]) Reserve(n uintptr) ([]T, error) {
	if n == 0 {
		return a.raw[:0], nil
	}
	start := a.count.Add(n) - n
	if start+n > a.maxElems {
		// rollback
		a.count.Add(^uintptr(n) + 1)
		return nil, ErrArenaFull
	}
	return a.raw[start : start+n], nil
}

// AppendSlice is now an alias for Reserve: it performs only an atomic reservation
// and returns a slice segment of length len(objs). No copy is done.
// To fill the arena, copy into the returned slice manually.
func (a *AtomicArena[T]) AppendSlice(objs []T) ([]T, error) {
	n := uintptr(len(objs))
	// Reserve raw slots
	seg, err := a.Reserve(n)
	if err != nil {
		return nil, err
	}
	// Copy input values into reserved segment
	copy(seg, objs)
	return seg, nil
}

//go:linkname memclrNoHeapPointers runtime.memclrNoHeapPointers
//go:nosplit
func memclrNoHeapPointers(ptr unsafe.Pointer, n uintptr)

// Reset clears all published pointers, allowing reuse of the arena.
// It zeroes the ptrs slice via memclrNoHeapPointers and resets the allocation count.
func (a *AtomicArena[T]) Reset(release bool) {
	if release {
		a.Free()
	}
	a.count.Store(0)
}

// Free clears all published pointers and zeroes the raw storage.
func (a *AtomicArena[T]) Free() {
	old := a.count.Load()
	if old > 0 {
		// clear published pointers
		ptr := unsafe.Pointer(&a.ptrs[0])
		sz := unsafe.Sizeof(a.ptrs[0])
		memclrNoHeapPointers(ptr, old*sz)

		// **also** zero out raw storage:
		var zero T
		for i := uintptr(0); i < old; i++ {
			a.raw[i] = zero
		}
	}
}
