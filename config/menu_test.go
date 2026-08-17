package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/config"
)

// runMenu drives the menu with scripted input and returns what it drew.
func runMenu(t *testing.T, path, input string) string {
	t.Helper()

	menu, err := config.NewMenu(path)
	require.NoError(t, err)

	var out bytes.Buffer
	menu.SetIO(strings.NewReader(input), &out)

	require.NoError(t, menu.Run())

	return out.String()
}

func TestMenuListsEverySection(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	output := runMenu(t, path, "q\n")

	assert.Contains(t, output, "client")
	assert.Contains(t, output, "server")
	assert.Contains(t, output, "agent")
	assert.Contains(t, output, "signing_key")
	// An unset value reads as unset rather than as a blank line.
	assert.Contains(t, output, "(not set)")
}

func TestMenuEditsAndWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// Field 1 is client.server.
	runMenu(t, path, "1\nhttps://ci.example.com\nw\n")

	saved, err := config.LoadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "https://ci.example.com", saved.Client.Server)

	// The document can hold a signing key, so it is written for its
	// owner only.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestMenuQuitDiscardsUnsavedChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".atkins", "config.yml")

	// Edit, then quit and confirm the discard.
	runMenu(t, path, "1\nhttps://ci.example.com\nq\ny\n")

	assert.NoFileExists(t, path)
}

func TestMenuRefusesAnInvalidValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// Field 6 is client.timeout, a duration.
	output := runMenu(t, path, "6\nnot-a-duration\nq\n")

	assert.Contains(t, output, "must be a duration")
}

func TestMenuRejectsAnOutOfRangeChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	output := runMenu(t, path, "999\nq\n")

	assert.Contains(t, output, "pick a number between")
}

func TestMenuMarksEnvironmentOverrides(t *testing.T) {
	t.Setenv("ATKINS_SERVER", "https://from-env.example.com")

	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")
	output := runMenu(t, path, "q\n")

	// Editing a field the environment overrides would look like it did
	// nothing, so the menu says so.
	assert.Contains(t, output, "ATKINS_SERVER")
}

func TestMenuWriteRefusesAnInvalidDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// Field 7 is server.addr, which without a port cannot work; writing
	// must not silently persist it.
	output := runMenu(t, path, "7\nlocalhost\nw\nq\ny\n")

	assert.Contains(t, output, "must include a port")
	assert.NoFileExists(t, path)
}

func TestMenuOnAnExistingDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".atkins", "config.yml")

	cfg, err := config.Default()
	require.NoError(t, err)
	cfg.Client.Server = "https://existing.example.com"
	require.NoError(t, cfg.Save(path))

	output := runMenu(t, path, "q\n")
	assert.Contains(t, output, "https://existing.example.com")
}

func TestMenuEndsOnExhaustedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// A piped stdin that runs out means "done", not an error loop.
	menu, err := config.NewMenu(path)
	require.NoError(t, err)

	var out bytes.Buffer
	menu.SetIO(strings.NewReader(""), &out)

	assert.NoError(t, menu.Run())
}
