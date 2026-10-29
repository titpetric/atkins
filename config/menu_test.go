package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

// runMenu drives the menu with scripted input and returns what it drew.
func runMenu(t *testing.T, path, input string) string {
	t.Helper()

	menu, err := config.NewMenu(path)
	assert.NoError(t, err)

	var out bytes.Buffer
	menu.SetIO(&out, strings.NewReader(input))

	assert.NoError(t, menu.Run())

	return out.String()
}

func TestMenuListsEverySection(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	output := runMenu(t, path, "q\n")

	assert.Contains(t, output, "storage")
	assert.Contains(t, output, "database")
	assert.Contains(t, output, "record")
	// An unset value reads as unset and never as a blank line.
	assert.Contains(t, output, "(not set)")
}

func TestMenuEditsAndWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// Field 1 is storage.database.
	runMenu(t, path, "1\n/tmp/events.db\nw\n")

	saved, err := config.LoadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/events.db", saved.Storage.Database)

	// Save writes for its owner only: the document sits in a checkout
	// other people may have a copy of.
	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestMenuQuitDiscardsUnsavedChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".atkins", "config.yml")

	// Edit, then quit and confirm the discard.
	runMenu(t, path, "1\n/tmp/events.db\nq\ny\n")

	assert.NoFileExists(t, path)
}

func TestMenuRefusesAnInvalidValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// Field 2 is storage.record, a boolean.
	output := runMenu(t, path, "2\nmaybe\nq\n")

	assert.Contains(t, output, "must be true or false")
}

func TestMenuRejectsAnOutOfRangeChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	output := runMenu(t, path, "999\nq\n")

	assert.Contains(t, output, "pick a number between")
}

func TestMenuMarksEnvironmentOverrides(t *testing.T) {
	t.Setenv("ATKINS_DB", "/from-env.db")

	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")
	output := runMenu(t, path, "q\n")

	// The menu names the variable that sets a field from the
	// environment.
	assert.Contains(t, output, "ATKINS_DB")
}

// The menu validates the whole document before writing it. No field it
// lists can make the document invalid - a path is a string and recording
// is a boolean - so TestValidateRejectsAVersionFromTheFuture tests the
// refusal, and this test the write the menu performs.
func TestMenuWritesAValidDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	runMenu(t, path, "2\nfalse\nw\n")

	saved, err := config.LoadFile(path)
	assert.NoError(t, err)
	assert.False(t, saved.Storage.Record)
}

func TestMenuOnAnExistingDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".atkins", "config.yml")

	cfg, err := config.Default()
	assert.NoError(t, err)
	cfg.Storage.Database = "/existing.db"
	assert.NoError(t, cfg.Save(path))

	output := runMenu(t, path, "q\n")
	assert.Contains(t, output, "/existing.db")
}

func TestMenuEndsOnExhaustedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	// A piped stdin that runs out means "done" and never loops on errors.
	menu, err := config.NewMenu(path)
	assert.NoError(t, err)

	var out bytes.Buffer
	menu.SetIO(&out, strings.NewReader(""))

	assert.NoError(t, menu.Run())
}
