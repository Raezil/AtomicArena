<p align="center">
  <img src="https://github.com/user-attachments/assets/eaf20af4-1108-4376-a781-1258c4a7b7fc">
</p>

# atomicArena

atomicArena` is a high-performance, concurrent-safe, fixed-size ring allocator for Go generics. It provides a simple API to allocate values in a ring buffer with minimal locking and low garbage collection overhead.

## Features

- **Generic**: Works with any type `T` using Go 1.18+ generics
- **Fixed-size ring**: Bump-pointer allocation into a circular buffer of predefined capacity
- **Concurrent-safe**: Internal mutex and atomic counter ensure thread-safe operations
- **Low GC overhead**: `Alloc` returns a fresh pointer to a copy of the value, avoiding reuse conflicts

## Installation

```bash
go get github.com/yourusername/atomicArena
```

Replace `github.com/yourusername/atomicArena` with the import path for your module.

## Usage

```go
package main

import (
    "fmt"
    "github.com/yourusername/atomicArena"
)

func main() {
    // Create an arena of 1024 slots for int values
    arena := atomicArena.NewAtomicArena[int](1024)

    // Allocate a value; Alloc returns a pointer to an independent copy
    ptr := arena.Alloc(42)
    fmt.Println("Allocated value:", *ptr)

    // Peek at the slot at index 0 (mod 1024)
    old := arena.PtrAt(0)
    fmt.Println("Value in ring at index 0:", *old)
}
```

## API Reference

### `func NewAtomicArena[T any](size int) *AtomicArena[T]`

- **Description**: Creates a new `AtomicArena` with exactly `size` slots.
- **Parameters**:
  - `size int`: Number of slots in the ring. Must be > 0.
- **Panics**: If `size <= 0`.
- **Returns**: A pointer to an `AtomicArena[T]`.

### `func (a *AtomicArena[T]) Alloc(val T) *T`

- **Description**: Atomically grabs the next slot in the ring, writes `val` into the buffer, and returns a pointer to an independent copy of `val`.
- **Parameters**:
  - `val T`: The value to allocate.
- **Returns**: `*T` pointer to the allocated copy.

### `func (a *AtomicArena[T]) PtrAt(i uint64) *T`

- **Description**: Retrieves a pointer to the value stored in the ring at index `i mod size` without synchronization.
- **Parameters**:
  - `i uint64`: The index to peek at. Wrap-around is handled via modulo.
- **Returns**: `*T` pointer to the value in the ring buffer.

## Concurrency

`AtomicArena` uses an internal `sync.Mutex` to guard concurrent allocations and an atomic counter to advance the bump pointer. While `Alloc` is safe for concurrent use, `PtrAt` is not synchronized and should be used when occasional race conditions on peeks are acceptable.
