package treeview

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"

	"github.com/titpetric/atkins/colors"
)

// Display renders the tree in place with ANSI cursor control. On a terminal
// each Render rolls back the lines of the previous one; Invalidate forgets
// them when output from outside the tree, such as an interactive command,
// lands between two renders. RenderFinal prints the whole tree once execution ends, so
// the output stays scrollable. A silent display, for JSON and YAML output,
// prints nothing live.
type Display struct {
	lastLineCount int
	mu            sync.Mutex
	isTerminal    bool
	renderer      *Renderer
	finalOnly     bool
}

// NewDisplay creates a new display manager.
func NewDisplay() *Display {
	return &Display{
		lastLineCount: 0,
		isTerminal:    colors.StdoutIsTerminal(),
		renderer:      NewRenderer(),
		finalOnly:     false,
	}
}

// NewDisplayWithFinal creates a new display manager with final-only mode.
func NewDisplayWithFinal(finalOnly bool) *Display {
	return &Display{
		lastLineCount: 0,
		isTerminal:    colors.StdoutIsTerminal() && !finalOnly,
		renderer:      NewRenderer(),
		finalOnly:     finalOnly,
	}
}

// NewSilentDisplay returns a display that renders nothing live.
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

// Invalidate resets the line counter, so the next Render rolls nothing back.
func (d *Display) Invalidate() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastLineCount = 0
}

// Render outputs the tree, updating in-place after the first render.
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
	fmt.Print(plainWithoutTerminal(output))
}

// RenderFinal clears the live tree and prints the full tree statically.
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
	fmt.Print(plainWithoutTerminal(output))
}

// plainWithoutTerminal strips ANSI escape sequences when stdout is a
// pipe or a file, so colour follows terminal detection and a captured
// log keeps its escape sequences out.
func plainWithoutTerminal(output string) string {
	if colors.StdoutIsTerminal() {
		return output
	}
	return colors.StripANSI(output)
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
