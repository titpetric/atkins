package treeview

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/term"

	"github.com/titpetric/atkins/colors"
)

// Trimmer handles label and line trimming for viewport constraints.
type Trimmer struct {
	viewportWidth int
	mu            sync.RWMutex
}

// NewTrimmer creates a new trimmer with detected viewport width.
func NewTrimmer() *Trimmer {
	t := &Trimmer{}
	t.detectViewport()
	return t
}

// detectViewport updates the viewport width from terminal size.
func (t *Trimmer) detectViewport() {
	t.mu.Lock()
	defer t.mu.Unlock()

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		t.viewportWidth = 0 // No trimming if we can't detect
		return
	}
	t.viewportWidth = width
}

// RefreshViewport re-detects the terminal width (call before each render).
func (t *Trimmer) RefreshViewport() {
	t.detectViewport()
}

// GetViewportWidth returns the current viewport width.
func (t *Trimmer) GetViewportWidth() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.viewportWidth
}

// SetViewportWidth sets a custom viewport width (useful for testing).
func (t *Trimmer) SetViewportWidth(width int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.viewportWidth = width
}

// argPattern matches flag=value, handling both quoted and unquoted values.
// Matches: -flag=value, --flag="value", -flag=123, etc.
var argPattern = regexp.MustCompile(`(-+[\w-]+=)("[^"]*"|[^\s]+)`)

// CompactArgs trims long argument values in a command string.
// Arguments longer than maxArgLen are replaced with <...N chars>.
func CompactArgs(cmd string, maxArgLen int) string {
	if maxArgLen <= 0 {
		return cmd
	}

	return argPattern.ReplaceAllStringFunc(cmd, func(match string) string {
		parts := argPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		flag, valueWithQuotes := parts[1], parts[2]

		// Strip quotes for length calculation
		value := valueWithQuotes
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		if len(value) <= maxArgLen {
			return match
		}

		return flag + "<..." + strconv.Itoa(len(value)) + " chars>"
	})
}

// TrimToViewport trims a line to fit within the viewport width.
// It accounts for ANSI escape codes and adds "..." suffix when trimmed.
// The prefixLen parameter indicates how many visual characters of prefix
// (indentation, branch characters) are already used.
func (t *Trimmer) TrimToViewport(line string, prefixLen int) string {
	t.mu.RLock()
	width := t.viewportWidth
	t.mu.RUnlock()

	if width <= 0 {
		return line // No viewport constraint
	}

	availableWidth := width - prefixLen
	if availableWidth <= 3 {
		return line // Not enough space to trim meaningfully
	}

	visualLen := colors.VisualLength(line)
	if visualLen <= availableWidth {
		return line // Fits within viewport
	}

	// We need to trim. Find the cut point in the original string
	// accounting for ANSI escape codes.
	return trimWithANSI(line, availableWidth-3) + "..."
}

// trimWithANSI trims a string to a visual length, preserving ANSI codes.
func trimWithANSI(s string, targetLen int) string {
	if targetLen <= 0 {
		return ""
	}

	var result strings.Builder
	visualCount := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}

		if inEscape {
			result.WriteRune(r)
			// End of escape sequence
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}

		// Regular visible character
		if visualCount >= targetLen {
			break
		}
		result.WriteRune(r)
		visualCount++
	}

	// Append reset code if we were in the middle of colored text
	result.WriteString("\033[0m")

	return result.String()
}

// TrimLabel applies both compaction and viewport trimming to a label.
// - maxArgLen: maximum length for argument values before compaction
// - prefixLen: visual length of the prefix (indentation, branch chars).
func (t *Trimmer) TrimLabel(label string, maxArgLen, prefixLen int) string {
	// First, compact arguments
	compacted := CompactArgs(label, maxArgLen)

	// Then trim to viewport
	return t.TrimToViewport(compacted, prefixLen)
}
