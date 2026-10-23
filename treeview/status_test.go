package treeview_test

import (
	"testing"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/treeview"
)

// TestStatus_String checks each status draws its indicator, and an unknown
// one draws nothing.
func TestStatus_String(t *testing.T) {
	t.Parallel()

	want := map[treeview.Status]string{
		treeview.StatusPending:     "●",
		treeview.StatusRunning:     "●",
		treeview.StatusPassed:      "✓",
		treeview.StatusFailed:      "✗",
		treeview.StatusSkipped:     "⊘",
		treeview.StatusConditional: "●",
		treeview.Status(99):        "",
	}
	for status, indicator := range want {
		assert.Equal(t, indicator, colors.StripANSI(status.String()))
	}
	assert.Equal(t, "passed", treeview.StatusPassed.Label())
	assert.Equal(t, "unknown", treeview.Status(99).Label())
}
