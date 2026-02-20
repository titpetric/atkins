package eventlog

import (
	"runtime"
)

// RuntimeStats holds memory and goroutine statistics.
type RuntimeStats struct {
	MemoryAlloc uint64
	Goroutines  int
}

// CaptureRuntimeStats captures current memory allocation and goroutine count.
func CaptureRuntimeStats() RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return RuntimeStats{
		MemoryAlloc: m.Alloc,
		Goroutines:  runtime.NumGoroutine(),
	}
}
