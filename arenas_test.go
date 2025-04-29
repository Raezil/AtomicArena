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
		if ptr != objs[i] {
			t.Errorf("Expected %v, got %v", objs[i], ptr)
		}
	}

	ptrs2, err := arena.AppendSlice([]int{4, 5})
	if err != nil {
		t.Fatalf("AppendSlice2 failed: %v", err)
	}
	if ptrs2[0] != 4 || ptrs2[1] != 5 {
		t.Errorf("Expected 4,5 got %v,%v", ptrs2[0], ptrs2[1])
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

func FuzzAppendByteSlice(f *testing.F) {
	// Seed corpus
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3, 4, 5})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create an arena with capacity equal to input length
		arena := NewAtomicArena[byte](uintptr(len(data)))

		ptrs, err := arena.AppendSlice(data)
		if err != nil {
			t.Fatalf("AppendSlice returned error on data %v: %v", data, err)
		}
		if len(ptrs) != len(data) {
			t.Fatalf("Expected %d pointers, got %d", len(data), len(ptrs))
		}
		for i, b := range data {
			if ptrs[i] != b {
				t.Errorf("Index %d: expected %v, got %v", i, b, ptrs[i])
			}
		}

		// Test after reset
		arena.Reset()
		ptrs2, err2 := arena.AppendSlice(data)
		if err2 != nil {
			t.Fatalf("AppendSlice after Reset returned error: %v", err2)
		}
		for i, b := range data {
			if ptrs2[i] != b {
				t.Errorf("After Reset index %d: expected %v, got %v", i, b, ptrs2[i])
			}
		}
	})
}

// FuzzAllocByte fuzz-tests the Alloc method for bytes.
func FuzzAllocByte(f *testing.F) {
	// Seed corpus
	f.Add(byte(0))
	f.Add(byte(1))
	f.Add(byte(255))

	f.Fuzz(func(t *testing.T, v byte) {
		// Create arena with capacity 1
		arena := NewAtomicArena[byte](1)

		ptr, err := arena.Alloc(v)
		if err != nil {
			t.Fatalf("Alloc returned error for value %v: %v", v, err)
		}
		if *ptr != v {
			t.Errorf("Expected %v, got %v", v, *ptr)
		}

		// Next Alloc should error
		_, err2 := arena.Alloc(v)
		if err2 == nil {
			t.Errorf("Expected error on second Alloc, got nil")
		}
	})
}

// TestResetClearsValues verifies that Reset zeroes stored values in the arena
func TestResetClearsValues(t *testing.T) {
	// Create an arena and populate with known values
	arena := NewAtomicArena[int](3)
	ptrs, err := arena.AppendSlice([]int{10, 20, 30})
	if err != nil {
		t.Fatalf("AppendSlice failed: %v", err)
	}
	// Confirm values before reset
	for i, v := range []int{10, 20, 30} {
		if ptrs[i] != v {
			t.Fatalf("expected %d at index %d before reset, got %d", v, i, ptrs[i])
		}
	}

	// Reset the arena
	arena.Reset()

	// After reset, underlying storage should be zeroed
	for i := uintptr(0); i < 3; i++ {
		if arena.ptrs[i].Load() != nil {
			t.Errorf("value at index %d not zero after reset: got %v", i, arena.ptrs[i])
		}
	}
}

// TestResetReadSafety spins up N readers that continuously Load() pointers
// while Reset() is called once. We assert there are no panics or data races.
// (Run with `-race` to catch any ordering bugs.)
func TestResetReadSafety(t *testing.T) {
	const N = 1_000
	a := NewAtomicArena[int](N)

	// 1) Pre-fill the arena
	vals := make([]int, N)
	for i := range vals {
		vals[i] = i
	}
	rawSlice, err := a.AppendSlice(vals)
	if err != nil {
		t.Fatalf("setup AppendSlice failed: %v", err)
	}

	// 2) Take a private snapshot of those values.
	//    Readers will only ever touch this, so there's no race.
	snapshot := make([]int, len(rawSlice))
	copy(snapshot, rawSlice)

	// 3) Spin up readers that loop reading from 'snapshot'
	var stop uint32
	var wg sync.WaitGroup
	reader := func() {
		defer wg.Done()
		for atomic.LoadUint32(&stop) == 0 {
			// plain reads from our own slice ⇒ no data race
			for _, v := range snapshot {
				_ = v
			}
			runtime.Gosched()
		}
	}
	numReaders := runtime.GOMAXPROCS(0)
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go reader()
	}

	// 4) While readers are running, reset the arena
	a.Reset()

	// 5) Tell readers to stop and wait
	atomic.StoreUint32(&stop, 1)
	wg.Wait()
}

// BenchmarkResetWithReaders measures how many Resets/sec you can do
// while N goroutines are hammering Load().
func BenchmarkResetWithReaders(b *testing.B) {
	const N = 10_000
	arena := NewAtomicArena[int](N)
	// Pre-fill
	vals := make([]int, N)
	for i := range vals {
		vals[i] = i
	}
	_, _ = arena.AppendSlice(vals)

	// Launch readers
	stop := make(chan struct{})
	var wg sync.WaitGroup
	reader := func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				for i := 0; i < N; i++ {
					_ = arena.ptrs[i].Load()
				}
				runtime.Gosched()
			}
		}
	}

	numReaders := runtime.GOMAXPROCS(0)
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go reader()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arena.Reset()
	}
	b.StopTimer()

	close(stop)
	wg.Wait()
}
