package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/runner"
)

// TestFindRepositoryRoot covers locating the folder that holds .git.
func TestFindRepositoryRoot(t *testing.T) {
	t.Run("finds root from a subdirectory", func(t *testing.T) {
		root, _ := vendorFixture(t)
		sub := filepath.Join(root, "server", "web")
		assert.NoError(t, os.MkdirAll(sub, 0o755))

		found, err := runner.FindRepositoryRoot(sub)
		assert.NoError(t, err)
		assert.Equal(t, root, found)
	})

	t.Run("finds root when .git is a file", func(t *testing.T) {
		base := t.TempDir()
		writeFile(t, filepath.Join(base, ".git"), "gitdir: /elsewhere/.git/worktrees/x\n")

		found, err := runner.FindRepositoryRoot(base)
		assert.NoError(t, err)
		assert.Equal(t, base, found)
	})

	t.Run("errors outside a repository", func(t *testing.T) {
		_, err := runner.FindRepositoryRoot(t.TempDir())
		assert.Error(t, err)
	})
}
