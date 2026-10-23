package eventlog_test

import (
	"testing"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
)

func TestCaptureRuntimeStats(t *testing.T) {
	t.Parallel()

	var stats eventlog.RuntimeStats = eventlog.CaptureRuntimeStats()
	assert.True(t, stats.MemoryAlloc > 0)
	assert.True(t, stats.Goroutines >= 1)
}
