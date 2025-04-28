package atomicarena

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

// TestAllocInt ensures Alloc succeeds when within capacity for int
func TestAllocInt(t *testing.T) {
	// Create arena for exactly one int (size 8 bytes on 64-bit)
	arena := NewAtomicArena[int](1)
	val, err := arena.Alloc(42)
	if err != nil {
		t.Fatalf("Alloc returned error: %v", err)
	}
	if *val != 42 {
		t.Errorf("expected 42, got %d", *val)
	}
}

// TestAllocStruct ensures Alloc succeeds for a struct and errors when exceeding capacity
func TestAllocStruct(t *testing.T) {
	type S struct{ A, B int64 }
	// size of S is 16 bytes, so maxElems=1 gives capacity 16
	arena := NewAtomicArena[S](1)
	_, err := arena.Alloc(S{1, 2})
	if err != nil {
		t.Fatalf("Alloc returned error: %v", err)
	}
	// Second alloc should fail
	_, err = arena.Alloc(S{3, 4})
	if err == nil {
		t.Fatal("expected error on second alloc, got none")
	}
}

// TestReset ensures Reset allows reusing arena after clearing
func TestReset(t *testing.T) {
	arena := NewAtomicArena[int](1)
	_, err := arena.Alloc(7)
	if err != nil {
		t.Fatalf("initial alloc failed: %v", err)
	}
	arena.Reset()
	// After reset, should be able to alloc again
	_, err = arena.Alloc(8)
	if err != nil {
		t.Fatalf("Alloc after reset failed: %v", err)
	}
}

// TestConcurrentAlloc tests concurrent allocations up to capacity
func TestConcurrentAlloc(t *testing.T) {
	arena := NewAtomicArena[int](10)
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

	_, err := arena.Alloc(99)
	if err == nil {
		t.Fatal("expected error after capacity reached, got none")
	}
}

// benchSizes defines total buffer sizes from 100 B up to 100 MB
var benchSizes = []struct {
	name       string
	totalBytes uintptr
}{
	{"100B", 100},
	{"1KB", 1 << 10},
	{"10KB", 10 << 10},
	{"100KB", 100 << 10},
	{"1MB", 1 << 20},
	{"10MB", 10 << 20},
	{"100MB", 100 << 20},
}

// BenchmarkReset measures the cost of Reset() for arenas of different buffer sizes.
func BenchmarkReset(b *testing.B) {
	for _, s := range benchSizes {
		s := s
		b.Run(s.name, func(b *testing.B) {
			// Each atomic.Pointer[T] is the size of an unsafe.Pointer
			pointerSize := unsafe.Sizeof(atomic.Pointer[struct{}]{})
			maxElems := s.totalBytes / pointerSize
			arena := NewAtomicArena[struct{}](maxElems)

			// Prefill all slots so Reset has to clear them
			for i := uintptr(0); i < maxElems; i++ {
				arena.Alloc(struct{}{})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				arena.Reset()
			}
		})
	}
}

// BenchmarkAlloc measures Alloc() throughput, resetting when full.
func BenchmarkAlloc(b *testing.B) {
	for _, s := range benchSizes {
		s := s
		b.Run(s.name, func(b *testing.B) {
			pointerSize := unsafe.Sizeof(atomic.Pointer[int]{})
			maxElems := s.totalBytes / pointerSize
			arena := NewAtomicArena[int](maxElems)

			b.ResetTimer()
			for i := 0; i < b.N; {
				_, err := arena.Alloc(i)
				if err != nil {
					arena.Reset()
					continue
				}
				i++
			}
		})
	}
}

func TestAppendSlice(t *testing.T) {
	arena := NewAtomicArena[int](5)
	objs := []int{1, 2, 3}
	ptrs, err := arena.AppendSlice(objs)
	if err != nil {
		t.Fatalf("AppendSlice failed: %v", err)
	}
	if len(ptrs) != len(objs) {
		t.Fatalf("Expected %v pointers, got %v", len(objs), len(ptrs))
	}
	for i, ptr := range ptrs {
		if *ptr != objs[i] {
			t.Errorf("Expected %v, got %v", objs[i], *ptr)
		}
	}

	ptrs2, err := arena.AppendSlice([]int{4, 5})
	if err != nil {
		t.Fatalf("AppendSlice2 failed: %v", err)
	}
	if *ptrs2[0] != 4 || *ptrs2[1] != 5 {
		t.Errorf("Expected 4,5 got %v,%v", *ptrs2[0], *ptrs2[1])
	}

	_, err = arena.AppendSlice([]int{6})
	if err == nil {
		t.Errorf("Expected error on full arena, got nil")
	}
}

func BenchmarkAppendSlice(b *testing.B) {
	size := uintptr(1000)
	objs := make([]int, size)
	for i := range objs {
		objs[i] = i
	}
	arena := NewAtomicArena[int](size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arena.Reset()
		_, _ = arena.AppendSlice(objs)
	}
}

// nativeBenchSizes lists the target allocation sizes from 100 B up to 100 MB.
var nativeBenchSizes = []struct {
	name string
	size int
}{
	{"100B", 100},
	{"1KB", 1 << 10},
	{"10KB", 10 << 10},
	{"100KB", 100 << 10},
	{"1MB", 1 << 20},
	{"10MB", 10 << 20},
	{"100MB", 100 << 20},
}

// BenchmarkNativeMake measures the cost of allocating a []byte of various sizes via make.
func BenchmarkNativeMake(b *testing.B) {
	for _, s := range nativeBenchSizes {
		s := s
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = make([]byte, s.size)
			}
		})
	}
}

// Define fixed-size array types so we can use new() for each target size.
type (
	buf100B  [100]byte
	buf1KB   [1 << 10]byte
	buf10KB  [10 << 10]byte
	buf100KB [100 << 10]byte
	buf1MB   [1 << 20]byte
	buf10MB  [10 << 20]byte
	buf100MB [100 << 20]byte
)

// bufAllocFuncs maps each size to a function that calls new() on the corresponding array type.
var bufAllocFuncs = []struct {
	name  string
	alloc func() interface{}
}{
	{"100B", func() interface{} { return new(buf100B) }},
	{"1KB", func() interface{} { return new(buf1KB) }},
	{"10KB", func() interface{} { return new(buf10KB) }},
	{"100KB", func() interface{} { return new(buf100KB) }},
	{"1MB", func() interface{} { return new(buf1MB) }},
	{"10MB", func() interface{} { return new(buf10MB) }},
	{"100MB", func() interface{} { return new(buf100MB) }},
}

// BenchmarkNativeNew measures the cost of allocating a fixed-size array via new().
func BenchmarkNativeNew(b *testing.B) {
	var sink interface{}
	for _, entry := range bufAllocFuncs {
		entry := entry
		b.Run(entry.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Store the result to prevent optimization
				sink = entry.alloc()
			}
			// Use sink in some way to prevent the compiler from optimizing it away
			runtime.KeepAlive(sink)
		})
	}
}
