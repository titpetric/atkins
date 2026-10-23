package version_test

import (
	"bytes"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/version"
)

func TestInfo(t *testing.T) {
	t.Parallel()

	info := version.Info{Version: "v1.2.3", Commit: "unknown", CommitTime: "unknown", Branch: "unknown"}

	var out bytes.Buffer
	assert.NoError(t, version.Run(&out, info))

	// An unknown value prints no line of its own.
	assert.Contains(t, out.String(), "  Version:     v1.2.3\n")
	assert.NotContains(t, out.String(), "Commit:")
	assert.NotContains(t, out.String(), "Branch:")
}

func TestRun(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := version.Run(&out, version.Info{
		Version:    "v1.2.3",
		Commit:     "0123456789abcdef",
		CommitTime: "2026-10-23T10:00:00Z",
		Branch:     "main",
	})
	assert.NoError(t, err)

	// The commit shortens to twelve characters.
	assert.Contains(t, out.String(), "  Commit:      0123456789ab\n")
	assert.NotContains(t, out.String(), "0123456789abc")
	assert.Contains(t, out.String(), "  CommitTime:  2026-10-23T10:00:00Z\n")
	assert.Contains(t, out.String(), "  Branch:      main\n")
}
