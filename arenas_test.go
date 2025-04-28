package atomicarena

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"
)

type MyStruct struct {
	A, B int64
}

type LargeArray [1024]byte

type HugeArray [1 << 20]byte // ~1MB

func TestAllocInt(t *testing.T) {
	mem := &AtomicArena[int]{}
	const N = 5
	var sumOffsets uintptr

	for i := 0; i < N; i++ {
		ptr, err := mem.Alloc(i * 10)
		if err != nil {
			t.Fatalf("Alloc returned error: %v", err)
		}
		if ptr == nil {
			t.Fatal("Alloc returned nil pointer")
		}
		if *ptr != i*10 {
			t.Errorf("got %d; want %d", *ptr, i*10)
		}
		sumOffsets += unsafe.Sizeof(i)
	}

	if len(mem.buff) != N {
		t.Errorf("buff length = %d; want %d", len(mem.buff), N)
	}
	if mem.offset != sumOffsets {
		t.Errorf("offset = %d; want %d", mem.offset, sumOffsets)
	}
}

func TestAllocStruct(t *testing.T) {
	mem := &AtomicArena[MyStruct]{}
	val := MyStruct{A: 42, B: 99}
	ptr, err := mem.Alloc(val)
	if err != nil {
		t.Fatalf("Alloc returned error: %v", err)
	}
	if ptr == nil {
		t.Fatal("Alloc returned nil pointer")
	}
	if *ptr != val {
		t.Errorf("got %+v; want %+v", *ptr, val)
	}

	expectedOffset := unsafe.Sizeof(val)
	if mem.offset != expectedOffset {
		t.Errorf("offset = %d; want %d", mem.offset, expectedOffset)
	}
}

func TestReset(t *testing.T) {
	mem := &AtomicArena[int]{}
	// allocate a few items
	for i := 0; i < 3; i++ {
		if _, err := mem.Alloc(i); err != nil {
			t.Fatalf("Alloc failed: %v", err)
		}
	}
	// now reset
	mem.Reset()
	if mem.offset != 0 {
		t.Errorf("after Reset, offset = %d; want 0", mem.offset)
	}
	for i, ap := range mem.buff {
		if got := ap.Load(); got != nil {
			t.Errorf("buff[%d].Load() = %v; want nil", i, got)
		}
	}
}

// Benchmarks for int allocations
func BenchmarkArenaAllocInt(b *testing.B) {
	mem := &AtomicArena[int]{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.Alloc(i)
	}
}

func BenchmarkNewInt(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := new(int)
		*ptr = i
		_ = ptr
	}
}

// Benchmarks for ~1KB allocations
func BenchmarkArenaAllocLargeArray(b *testing.B) {
	mem := &AtomicArena[LargeArray]{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.Alloc(LargeArray{})
	}
}

func BenchmarkNewLargeArray(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := new(LargeArray)
		_ = ptr
	}
}

// Benchmarks for ~1MB allocations
func BenchmarkArenaAllocHugeArray(b *testing.B) {
	mem := &AtomicArena[HugeArray]{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.Alloc(HugeArray{})
	}
}

func BenchmarkNewHugeArray(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := new(HugeArray)
		_ = ptr
	}
}

// Test that Alloc succeeds when within capacity
func TestAllocWithinCapacity(t *testing.T) {
	// Use an arena with capacity for exactly one int
	arena := NewAtomicArena[int](unsafe.Sizeof(int(0)))
	val, err := arena.Alloc(100)
	if err != nil {
		t.Fatalf("unexpected error on first alloc: %v", err)
	}
	if *val != 100 {
		t.Errorf("expected allocated value 100, got %d", *val)
	}
}

// Test that Alloc returns error when exceeding capacity
func TestAllocExceedCapacity(t *testing.T) {
	// Capacity only enough for one int
	arena := NewAtomicArena[int](unsafe.Sizeof(int(0)))
	_, err := arena.Alloc(1)
	if err != nil {
		t.Fatalf("unexpected error on first alloc: %v", err)
	}
	// Second allocation should fail
	_, err = arena.Alloc(2)
	if err == nil {
		t.Fatal("expected error when arena is full, got none")
	}
	// Ensure error message mentions capacity and request size
	msg := err.Error()
	expected := fmt.Sprintf("arena full: capacity %d bytes exceeded by request of %d bytes", arena.capacity, unsafe.Sizeof(int(0)))
	if msg != expected {
		t.Errorf("unexpected error message: got %q, want %q", msg, expected)
	}
}

// Test concurrent usage
func TestConcurrentAlloc(t *testing.T) {
	arena := NewAtomicArena[int](unsafe.Sizeof(int(0)) * 10)
	count := 10

	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := arena.Alloc(i)
			if err != nil {
				t.Errorf("unexpected error on alloc %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// After 10 allocs, further alloc should fail
	_, err := arena.Alloc(99)
	if err == nil {
		t.Fatal("expected error after concurrent allocs exceed capacity, got none")
	}
}
