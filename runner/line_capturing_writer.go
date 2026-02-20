package runner

import (
	"bytes"
	"strings"
	"sync"
)

// LineCapturingWriter captures all output written to it.
type LineCapturingWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// NewLineCapturingWriter creates a new LineCapturingWriter.
func NewLineCapturingWriter() *LineCapturingWriter {
	return &LineCapturingWriter{}
}

// Write implements io.Writer.
func (w *LineCapturingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// GetLines returns all captured output as lines.
func (w *LineCapturingWriter) GetLines() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	output := w.buffer.String()
	if output == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	// Remove the last empty line if it exists (from final newline)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// String returns the raw captured output.
func (w *LineCapturingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}
