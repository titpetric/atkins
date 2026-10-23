package model_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestLabel checks the builders set the fields String and ForDisplay read.
func TestLabel(t *testing.T) {
	t.Parallel()

	label := model.NewLabel("go test", "run")
	assert.Equal(t, &model.Label{Text: "go test", Type: "run"}, label)
	assert.Equal(t, "go test", label.String())
	assert.Equal(t, "go test", label.ForDisplay())

	shout := func(s string) string { return "<" + s + ">" }
	label = label.WithPrefix(true).WithStatus("✓").WithColor(shout)
	assert.True(t, label.ShowPrefix)
	assert.Equal(t, "✓", label.Status)
	assert.Equal(t, "run: go test", label.String())
	assert.Equal(t, "<run: go test> ✓", label.ForDisplay())
}
