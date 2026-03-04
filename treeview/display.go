package treeview

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// Display manages in-place tree rendering with ANSI cursor control.
type Display struct {
	lastLineCount int
	mu            sync.Mutex
	isTerminal    bool
	renderer      *Renderer
	finalOnly     bool
}

// NewDisplay creates a new display manager.
func NewDisplay() *Display {
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	return &Display{
		lastLineCount: 0,
		isTerminal:    isTerminal,
		renderer:      NewRenderer(),
		finalOnly:     false,
	}
}

// NewDisplayWithFinal creates a new display manager with final-only mode.
func NewDisplayWithFinal(finalOnly bool) *Display {
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	return &Display{
		lastLineCount: 0,
		isTerminal:    isTerminal && !finalOnly,
		renderer:      NewRenderer(),
		finalOnly:     finalOnly,
	}
}

// NewSilentDisplay creates a display that produces no output.
// Used when JSON/YAML output mode is enabled.
func NewSilentDisplay() *Display {
	return &Display{
		lastLineCount: 0,
		isTerminal:    false,
		renderer:      NewRenderer(),
		finalOnly:     true,
	}
}

// IsTerminal returns whether stdout is a TTY.
func (d *Display) IsTerminal() bool {
	return d.isTerminal
}

// Invalidate resets the line counter so the next Render does not
// roll back over output that was produced outside the tree (e.g. by
// an interactive command).
func (d *Display) Invalidate() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastLineCount = 0
}

// Render outputs the tree, updating in-place if previously rendered.
func (d *Display) Render(root *Node) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Only render if stdout is a TTY (interactive terminal)
	if !d.isTerminal {
		return
	}

	output := d.renderer.Render(root)

	// Determine how many lines we can actually display. When the tree is
	// taller than the terminal, only show the bottom portion so the
	// rollback always matches what was previously on screen.
	_, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termHeight <= 0 {
		termHeight = 0 // unknown, no clamping
	}

	lines := strings.Split(output, "\n")
	// Remove trailing empty element from final \n
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	lineCount := len(lines)

	// Clamp to terminal height: keep only the bottom portion that fits
	if termHeight > 0 && lineCount > termHeight {
		lines = lines[lineCount-termHeight:]
		lineCount = termHeight
	}

	if d.lastLineCount > 0 {
		fmt.Printf("\033[%dA\033[J", d.lastLineCount)
	}

	for _, line := range lines {
		fmt.Print(line + "\n")
	}

	d.lastLineCount = lineCount
}

// RenderStatic displays a static tree view (for list).
func (d *Display) RenderStatic(root *Node) {
	d.mu.Lock()
	defer d.mu.Unlock()

	output := d.renderer.RenderStatic(root)
	fmt.Print(output)
}

// countOutputLines counts the number of newlines in output
func countOutputLines(output string) int {
	count := 0
	for _, ch := range output {
		if ch == '\n' {
			count++
		}
	}
	return count
}
