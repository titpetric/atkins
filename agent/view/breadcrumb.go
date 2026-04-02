package view

import (
	"strings"
	"time"
)

// Breadcrumb tracks execution progress as a one-liner.
type Breadcrumb struct {
	segments  []string
	status    string
	startTime time.Time
}

// NewBreadcrumb creates a new breadcrumb tracker.
func NewBreadcrumb() *Breadcrumb {
	return &Breadcrumb{
		segments: []string{},
	}
}

// Push adds a segment to the breadcrumb trail.
// e.g., "go" -> "go > test" -> "go > test > step 1".
func (b *Breadcrumb) Push(segment string) {
	if b.startTime.IsZero() {
		b.startTime = time.Now()
	}
	b.segments = append(b.segments, segment)
}

// Pop removes the last segment.
func (b *Breadcrumb) Pop() {
	if len(b.segments) > 0 {
		b.segments = b.segments[:len(b.segments)-1]
	}
}

// SetStatus updates the current status suffix.
// e.g., "running...", "passed", "failed".
func (b *Breadcrumb) SetStatus(status string) {
	b.status = status
}

// String renders the breadcrumb as a one-liner.
// Output: "go > test > step 1 [running...]".
func (b *Breadcrumb) String() string {
	if len(b.segments) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(b.segments, " > "))

	if b.status != "" {
		sb.WriteString(" [")
		sb.WriteString(b.status)

		// Add duration for completed states
		if !b.startTime.IsZero() && (b.status == "done" || b.status == "failed") {
			elapsed := time.Since(b.startTime)
			sb.WriteString(" ")
			sb.WriteString(FormatDuration(elapsed))
		}

		sb.WriteString("]")
	}

	return sb.String()
}

// Clear resets the breadcrumb.
func (b *Breadcrumb) Clear() {
	b.segments = b.segments[:0]
	b.status = ""
	b.startTime = time.Time{}
}

// LastSegment returns the last segment or empty string.
func (b *Breadcrumb) LastSegment() string {
	if len(b.segments) == 0 {
		return ""
	}
	return b.segments[len(b.segments)-1]
}

// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Minute {
		return d.Round(100 * time.Millisecond).String()
	}
	return d.Round(time.Second).String()
}
