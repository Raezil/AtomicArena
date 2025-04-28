# atomicarena

A lightweight, generic memory arena for Go, built with atomic pointers and `unsafe.Sizeof` for offset tracking. Perfect for scenarios where you want fast, lock-free allocations with the ability to reset the arena in one go.

## Features

- **Generic**: Works with any Go type (`T any`).
- **Atomic-safe**: Uses `sync/atomic.Pointer[T]` to store allocations, making it safe for concurrent reads.
- **Offset tracking**: Keeps a running total of bytes allocated via `unsafe.Sizeof`.
- **Easy reset**: `Reset()` zeroes out all stored pointers and resets the offset, preparing the arena for reuse.

## Installation

```bash
go get github.com/yourusername/atomicarena
```

> Replace `github.com/yourusername/atomicarena` with your module path.

## Usage

Import the package and start allocating objects in your arena:

```go
package main

import (
    "fmt"
    "github.com/yourusername/atomicarena"
)

func main() {
    // Create a new arena for ints
    arena := &atomicarena.AtomicArena[int]{}

    // Allocate some integers
    ptr1, _ := arena.Alloc(100)
    ptr2, _ := arena.Alloc(200)

    fmt.Println(*ptr1) // 100
    fmt.Println(*ptr2) // 200

    // Check total bytes allocated
    fmt.Printf("Total bytes: %d\n", arena.offset)

    // Reset the arena for reuse
    arena.Reset()
    fmt.Println("After reset, offset:", arena.offset)
}
```

For composite or custom types:

```go
package main

import (
    "fmt"
    "github.com/yourusername/atomicarena"
)

type Point struct { X, Y float64 }

func main() {
    arena := &atomicarena.AtomicArena[Point]{}
    p, _ := arena.Alloc(Point{X: 1.5, Y: 3.7})
    fmt.Printf("Allocated point: %+v\n", *p)
}
```

## API Reference

```go
// AtomicArena is a generic memory arena for type T.
type AtomicArena[T any] struct {
    buff   []atomic.Pointer[T] // underlying slice of pointers
    offset uintptr             // total bytes allocated
}

// Alloc stores a copy of obj in the arena and returns its pointer.
// Currently always returns a nil error.
func (mem *AtomicArena[T]) Alloc(obj T) (*T, error)

// Reset clears all stored pointers and resets the offset to zero.
func (mem *AtomicArena[T]) Reset()
```

## Running Tests & Benchmarks

```bash
go test -v ./atomicarena
go test -bench=. ./atomicarena
```

## Next Steps & Ideas 🚀

- **Pooling**: Integrate an object pool to reuse memory slots and reduce GC pressure.
- **Concurrent benchmarks**: Add `b.RunParallel` benches to measure real-world concurrent allocation performance.
- **GC impact**: Track GC metrics (pause times, collections) to validate arena benefits.

Enjoy exploring atomicarena! Contributions and feedback are welcome. Pull requests are always appreciated ❤️

