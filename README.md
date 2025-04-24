# AtomicArena

`AtomicArena[T]` is a fixed-size, bump-pointer ring allocator in Go, safe for concurrent use. It stores elements in a circular buffer and returns unique pointers to newly allocated values, avoiding data races.

## Installation

```sh
go get github.com/Raezil/atomicarena
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/Raezil/atomicarena"
)

type Item struct {
    Value int
}

func main() {
    // Create an arena with 100 slots
    arena, _ := atomicarena.NewAtomicArena[Item](100)

    // Allocate single items
    ptr, _ := arena.Alloc(Item{Value: 42})
    fmt.Println("Allocated value:", ptr.Value)

    // Allocate multiple values at once
    items := []Item{{1}, {2}, {3}}
    ptrs, _ := arena.AllocSlice(items)
    for _, p := range ptrs {
        fmt.Println("Batch Allocated:", p.Value)
    }

    // Build a full slice (fills missing slots with zero-values)
    fullSlice, _ := arena.MakeSlice()
    fmt.Printf("MakeSlice returned %d pointers\n", len(fullSlice))

    // Peek at a recent slot (e.g., first allocation)
    peek := arena.PtrAt(0)
    fmt.Println("Peeked value:", peek.Value)

    // Reset the arena
    arena.Reset()
    fmt.Println("After reset, peek slot 0 is nil?", arena.PtrAt(0) == nil)
}
```

## API

### `func NewAtomicArena[T any](size int) (*AtomicArena[T], error)`

Creates a new arena with the given number of slots (must be > 0). Returns an error if the size is invalid.

### `func (a *AtomicArena[T]) Alloc(val T) (*T, error)`

Allocates the next free slot, stores `val`, and returns a pointer to an independent copy. Returns an error if the arena is full.

### `func (a *AtomicArena[T]) AllocSlice(vals []T) ([]*T, error)`

Allocates each element from `vals` in sequence, returning a slice of pointers to the stored values. If the arena fills up before storing all elements, it returns an error and stops.

### `func (a *AtomicArena[T]) MakeSlice() ([]*T, error)`

Ensures the arena is completely filled by allocating zero-value elements for any remaining slots, then returns a slice of pointers to all stored values in allocation order. Returns an error if allocation fails during filling.

### `func (a *AtomicArena[T]) PtrAt(i uint64) *T`

Returns a pointer to the element at index `i mod size` in the ring buffer, allowing you to peek at past allocations.

### `func (a *AtomicArena[T]) Reset()`

Clears the arena back to its initial state:

- Locks the arena to prevent races with concurrent `Alloc`, `AllocSlice`, or `PtrAt` calls.
- Resets the internal allocation counter to zero.
- Zeroes out every slot in the buffer.

```go
func (a *AtomicArena[T]) Reset() {
    a.mu.Lock()
    defer a.mu.Unlock()

    a.counter = 0
    for i := range a.buf {
        a.buf[i].Store(nil)
    }
}
```

## Testing & Benchmarking

This package includes tests and benchmarks for `AllocSlice` and `MakeSlice`. To run them:

```sh
go test -bench=.
```

