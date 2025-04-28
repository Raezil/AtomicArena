# AtomicArena

`AtomicArena` is a lock-free, thread-safe generic arena allocator for Go. It provides a fixed-capacity container of atomic pointers, allowing you to allocate objects of any type **T** up to a maximum element count without locks or garbage collection overhead.

## Features

- **Generic**: Works with any Go type `T` whose size is known at compile time.
- **Lock-Free**: Uses atomic operations (`atomic.Uintptr`, `atomic.Pointer[T]`) for high concurrency without mutexes.
- **Fixed Capacity**: Pre-allocates a buffer for `maxElems` objects to avoid slice growth and dynamic allocations after initialization.
- **Resettable**: Clear the arena in constant time, reusing all slots immediately.

## Installation

```bash
go get github.com/Raezil/atomicarena
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/Raezil/atomicarena"
)

func main() {
    // Create an arena for 100 integers
    arena := atomicarena.NewAtomicArena[int](100)

    // Allocate a value
    ptr, err := arena.Alloc(42)
    if err != nil {
        panic(err)
    }
    fmt.Println("Allocated:", *ptr)

    // Reset to reuse all slots
    arena.Reset()
}
```

## API

### `NewAtomicArena[T any](maxElems uintptr) *AtomicArena[T]`
Creates a new arena capable of holding up to `maxElems` values of type `T`.

### `(a *AtomicArena[T]) Alloc(obj T) (*T, error)`
Atomically reserves a slot and stores `obj`. Returns an error if capacity is exhausted.

### `(a *AtomicArena[T]) Reset()`
Clears all allocations, setting the element count back to zero.

## Example: Structs

```go
// Define a struct
 type Point struct { X, Y float64 }

// Arena for 10 points
a := atomicarena.NewAtomicArena[Point](10)

// Allocate two points
p1, _ := a.Alloc(Point{1, 2})
p2, _ := a.Alloc(Point{3, 4})

fmt.Println(*p1, *p2)
```

## Testing

A comprehensive test suite covers:

- Allocating basic types (`int`, `string`, etc.) and structs
- Error on exceeding `maxElems`
- `Reset()` correctness
- High-concurrency allocations (data-race free)

Run tests with:

```bash
go test --bench=. --cover --race
```

