package main

import (
	"fmt"

	"github.com/Raezil/atomicarena"
)

func main() {
	// === Example 1: Entity Pooling ===
	type Entity struct{ X, Y float64 }
	const maxEntities = 3
	entityArena := atomicarena.NewAtomicArena[Entity](maxEntities)

	fmt.Println("-- Spawning Entities --")
	for i := 0; i < maxEntities; i++ {
		ptr, err := entityArena.Alloc(Entity{X: float64(i), Y: float64(i * 2)})
		if err != nil {
			fmt.Println("Alloc error:", err)
			break
		}
		fmt.Printf("Entity %d at (%.0f, %.0f)\n", i, ptr.X, ptr.Y)
	}
	// Attempt overflow
	if _, err := entityArena.Alloc(Entity{}); err != nil {
		fmt.Println("Overflow error:", err)
	}

	// Reset for reuse
	entityArena.Reset()
	fmt.Println("Entity arena reset. Next alloc() starts from 0 again.")
	// === Example 2: Packet Buffering ===
	type Packet struct{ Data []byte }
	const maxPackets = 2
	packetArena := atomicarena.NewAtomicArena[Packet](maxPackets)

	batch := []Packet{{Data: []byte("foo")}, {Data: []byte("barbaz")}}
	ptrs, err := packetArena.AppendSlice(batch)
	if err != nil {
		fmt.Println("AppendSlice error:", err)
	} else {
		fmt.Println("-- Buffered Packets --")
		for i, p := range ptrs {
			fmt.Printf("Packet %d length=%d\n", i, len(p.Data))
		}
	}

	// Reset and reuse
	packetArena.Reset()
	for i := range ptrs {
		fmt.Println(ptrs[i])
	}
	fmt.Println("Packet arena reset. Ready for next batch.")

	// === Example 3: Batch Logging ===
	type LogEntry struct{ Level, Msg string }
	const batchSize = 2
	logArena := atomicarena.NewAtomicArena[LogEntry](batchSize)

	entries := []LogEntry{{Level: "INFO", Msg: "Start work"}, {Level: "WARN", Msg: "Low memory"}}
	logPtrs, err := logArena.AppendSlice(entries)
	if err != nil {
		fmt.Println("AppendSlice error:", err)
	} else {
		fmt.Println("-- Log Batch --")
		for _, e := range logPtrs {
			fmt.Printf("[%s] %s\n", e.Level, e.Msg)
		}
	}

	// Final reset
	logArena.Reset()
	fmt.Println("Log arena reset.")
}
