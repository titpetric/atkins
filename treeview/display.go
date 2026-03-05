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

	if !d.isTerminal {
		return
	}

	_, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termHeight <= 0 {
		termHeight = 24
	}

	output := d.renderer.Render(root)
	lines := strings.Split(output, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Sliding window: show last (termHeight-1) lines to stay within terminal
	maxLines := termHeight - 1
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	// Roll back exactly what we printed last time (never more)
	if d.lastLineCount > 0 {
		fmt.Printf("\033[%dA\033[J", d.lastLineCount)
	}

	for _, line := range lines {
		fmt.Print(line + "\n")
	}

	d.lastLineCount = len(lines)
}

// RenderStatic displays a static tree view (for list).
func (d *Display) RenderStatic(root *Node) {
	d.mu.Lock()
	defer d.mu.Unlock()

	output := d.renderer.RenderStatic(root)
	fmt.Print(output)
}

// RenderFinal clears the live-updating tree and prints the full tree statically.
// This should be called when execution completes so the output is scrollable.
func (d *Display) RenderFinal(root *Node) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Roll back what we printed
	if d.isTerminal && d.lastLineCount > 0 {
		fmt.Printf("\033[%dA\033[J", d.lastLineCount)
		d.lastLineCount = 0
	}

	// Print the full tree
	output := d.renderer.Render(root)
	fmt.Print(output)
}

// Cleanup is a no-op kept for API compatibility.
func (d *Display) Cleanup() {}

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
