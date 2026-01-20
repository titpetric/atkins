package treeview

import "github.com/titpetric/atkins/colors"

// Status represents the execution status of a node.
type Status int

// Status constants.
const (
	StatusPending Status = iota
	StatusRunning
	StatusPassed
	StatusFailed
	StatusSkipped
	StatusConditional
)

// String returns a colored string representation of the Status for display.
func (s Status) String() string {
	switch s {
	case StatusPending:
		return colors.Gray("●")
	case StatusRunning:
		return colors.BrightOrange("●")
	case StatusPassed:
		return colors.BrightGreen("✓")
	case StatusFailed:
		return colors.BrightRed("✗")
	case StatusSkipped:
		return colors.BrightYellow("⊘")
	case StatusConditional:
		return colors.Gray("●")
	default:
	}
	return ""
}

// Label returns a lowercase readable label for the Status (for logging/serialization).
func (s Status) Label() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusPassed:
		return "passed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	case StatusConditional:
		return "conditional"
	default:
		return "unknown"
	}
}
