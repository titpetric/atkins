package runner_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/runner"
)

func TestLineCapturingWriter(t *testing.T) {
	t.Parallel()

	var w io.Writer = runner.NewLineCapturingWriter()
	_, err := fmt.Fprint(w, "one\ntwo\n")
	assert.NoError(t, err)
}

func TestNewLineCapturingWriter(t *testing.T) {
	t.Parallel()

	w := runner.NewLineCapturingWriter()
	assert.Nil(t, w.GetLines(), "nothing written, no lines")
	assert.Equal(t, "", w.String())
}

func TestLineCapturingWriter_Write(t *testing.T) {
	t.Parallel()

	w := runner.NewLineCapturingWriter()
	n, err := w.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
}

func TestLineCapturingWriter_GetLines(t *testing.T) {
	t.Parallel()

	w := runner.NewLineCapturingWriter()
	_, _ = w.Write([]byte("one\ntwo\n"))
	assert.Equal(t, []string{"one", "two"}, w.GetLines(), "the final newline adds no line")

	_, _ = w.Write([]byte("three"))
	assert.Equal(t, []string{"one", "two", "three"}, w.GetLines())
}

func TestLineCapturingWriter_String(t *testing.T) {
	t.Parallel()

	w := runner.NewLineCapturingWriter()
	_, _ = w.Write([]byte("one\ntwo\n"))
	assert.Equal(t, "one\ntwo\n", w.String())
}
