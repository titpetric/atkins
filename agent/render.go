package agent

import (
	"time"

	"github.com/titpetric/atkins/agent/view"
)

// RenderJobEntry renders a single job entry in gotestsum style.
func RenderJobEntry(name string, running, failed bool, duration time.Duration, errMsg string) string {
	return view.RenderJobEntry(name, running, failed, duration, errMsg)
}

// RenderJobSummary renders a summary line in gotestsum style.
func RenderJobSummary(total, passed, failed int, duration time.Duration) string {
	return view.RenderJobSummary(total, passed, failed, duration)
}
