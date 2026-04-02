package history_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/agent/history"
)

func TestShellHistory_NewShellHistory(t *testing.T) {
	h := history.NewShellHistory()
	assert.NotNil(t, h)
}

func TestShellHistory_Add(t *testing.T) {
	// Create a temporary home directory for testing
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .atkins directory
	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	h := history.NewShellHistory()
	h.Add("ls -la", 0, 100*time.Millisecond, "/home/user")

	matches := h.Match("ls")
	assert.Len(t, matches, 1)
	assert.Equal(t, "ls -la", matches[0].Command)
	assert.Equal(t, 0, matches[0].ExitCode)
}

func TestShellHistory_Match(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	h := history.NewShellHistory()
	h.Add("git status", 0, 50*time.Millisecond, "/repo")
	h.Add("git diff", 0, 60*time.Millisecond, "/repo")
	h.Add("git log", 0, 70*time.Millisecond, "/repo")
	h.Add("curl wttr.in", 0, 1*time.Second, "/home")

	// Match prefix
	matches := h.Match("git")
	assert.Len(t, matches, 3)

	// Most recent first
	assert.Equal(t, "git log", matches[0].Command)
	assert.Equal(t, "git diff", matches[1].Command)
	assert.Equal(t, "git status", matches[2].Command)

	// Match substring
	matches = h.Match("wttr")
	assert.Len(t, matches, 1)
	assert.Equal(t, "curl wttr.in", matches[0].Command)

	// No match
	matches = h.Match("xyz")
	assert.Empty(t, matches)
}

func TestShellHistory_FindExact(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	h := history.NewShellHistory()
	h.Add("echo hello", 0, 10*time.Millisecond, "/home")
	h.Add("echo world", 0, 10*time.Millisecond, "/home")
	h.Add("echo hello", 0, 10*time.Millisecond, "/home") // Duplicate

	// Find exact should return most recent
	entry := h.FindExact("echo hello")
	require.NotNil(t, entry)
	assert.Equal(t, "echo hello", entry.Command)

	// Not found
	entry = h.FindExact("not found")
	assert.Nil(t, entry)
}

func TestShellHistory_MatchLimit(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	h := history.NewShellHistory()

	// Add more than 10 commands
	for i := 0; i < 15; i++ {
		h.Add("cmd"+string(rune('a'+i)), 0, 10*time.Millisecond, "/home")
	}

	// Match should return at most 10
	matches := h.Match("cmd")
	assert.LessOrEqual(t, len(matches), 10)
}

func TestShellHistory_MatchDedupe(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	h := history.NewShellHistory()
	h.Add("echo test", 0, 10*time.Millisecond, "/home")
	h.Add("echo test", 0, 10*time.Millisecond, "/home")
	h.Add("echo test", 0, 10*time.Millisecond, "/home")

	// Should dedupe
	matches := h.Match("echo")
	assert.Len(t, matches, 1)
}
