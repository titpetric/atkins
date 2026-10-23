package main

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
)

func TestRunVendorJail(t *testing.T) {
	t.Parallel()

	err := runVendor(t.TempDir(), &Options{Vendor: true, Jail: true})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--jail")
}

func TestColorDiffLine(t *testing.T) {
	t.Parallel()

	for line, want := range map[string]string{
		"+++ b/go.yml": colors.Dim("+++ b/go.yml"),
		"--- a/go.yml": colors.Dim("--- a/go.yml"),
		"@@ -1,2 +1,3": colors.BrightCyan("@@ -1,2 +1,3"),
		"+added":       colors.BrightGreen("+added"),
		"-removed":     colors.BrightRed("-removed"),
		" context":     colors.Dim(" context"),
	} {
		assert.Equal(t, want, colorDiffLine(line), strings.TrimSpace(line))
	}
}
