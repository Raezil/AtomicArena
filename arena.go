package atomicarena

import (
	"sync/atomic"
	"unsafe"
)

type AtomicArena[T any] struct {
	buff   []atomic.Pointer[T]
	offset uintptr
}

func (mem *AtomicArena[T]) Alloc(obj T) (*T, error) {
	ptr := new(T)
	*ptr = obj

	var atomicPtr atomic.Pointer[T]
	atomicPtr.Store(ptr)

	mem.buff = append(mem.buff, atomicPtr)
	mem.offset += unsafe.Sizeof(obj)

	return ptr, nil
}

func (mem *AtomicArena[T]) Reset() {
	for i := range mem.buff {
		mem.buff[i].Store(nil)
	}
	mem.offset = 0
}
