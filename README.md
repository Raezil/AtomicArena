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
    arena := atomicarena.NewAtomicArena[Item](100)

    // Allocate a new item
    ptr := arena.Alloc(Item{Value: 42})
    fmt.Println("Allocated value:", ptr.Value)

    // Peek at a recent slot (e.g., first allocation)
    peek := arena.PtrAt(0)
    fmt.Println("Peeked value:", peek.Value)

    // Reset the arena
    arena.Reset()
    fmt.Println("After reset, peek at slot 0:", arena.PtrAt(0)) // zero value
}
```

## API

### `func NewAtomicArena[T any](size int) *AtomicArena[T]`

Creates a new ring arena of the given size (must be > 0). Panics otherwise.

### `func (a *AtomicArena[T]) Alloc(val T) *T`

Atomically allocates the next slot, overwrites it with `val`, and returns a pointer to an independent copy of that value.

### `func (a *AtomicArena[T]) PtrAt(i uint64) *T`

Returns a pointer to the element at index `i mod size` in the ring buffer, allowing you to peek at past allocations.

### `func (a *AtomicArena[T]) Reset()`

Clears the arena back to its initial state:

- **Locks** the arena to prevent races with concurrent `Alloc` or `PtrAt` calls.
- **Resets** the internal allocation counter to zero.
- **Zeroes** out every slot in the buffer.

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

