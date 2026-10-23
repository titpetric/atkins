package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
)

// TestHelpRequested covers the command lines that request the help
// document and the ones that carry -h or --help as a flag value.
func TestHelpRequested(t *testing.T) {
	flags := HelpFlags()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no arguments", args: nil, want: false},
		{name: "long flag", args: []string{"--help"}, want: true},
		{name: "short flag", args: []string{"-h"}, want: true},
		{name: "after a command", args: []string{"run", "--help"}, want: true},
		{name: "after a job name", args: []string{"go:fmt", "--help"}, want: true},
		{name: "after a boolean flag", args: []string{"--final", "-h"}, want: true},
		{name: "value of a short string flag", args: []string{"-w", "--help"}, want: false},
		{name: "value of a long string flag", args: []string{"--log", "--help"}, want: false},
		{name: "string flag with an inline value", args: []string{"--log=run.yml", "--help"}, want: true},
		{name: "after the argument separator", args: []string{"--", "--help"}, want: false},
		{name: "a job called help", args: []string{"help"}, want: false},
		{name: "unrelated flags", args: []string{"-l", "--json"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, HelpRequested(flags, test.args))
		})
	}
}

// TestHelpFlags checks the set carries the flags help detection has to
// know about, and tells a value flag from a boolean one.
func TestHelpFlags(t *testing.T) {
	flags := HelpFlags()

	assert.NotNil(t, flags.Lookup("list"))
	assert.NotNil(t, flags.Lookup("stats"))
	assert.NotNil(t, flags.Lookup("local"))

	assert.Equal(t, "bool", flags.Lookup("final").Value.Type())
	assert.Equal(t, "string", flags.Lookup("log").Value.Type())
}

// TestWriteHelp renders the document the binary prints and checks it
// carries the usage line, every command, the flags and the schema.
func TestWriteHelp(t *testing.T) {
	var out bytes.Buffer
	assert.NoError(t, WriteHelp(&out))

	doc := out.String()

	for _, want := range []string{
		"# atkins",
		"Usage: atkins [flags] [job...]",
		"## Commands",
		"`run`",
		"## Flags",
		"`-l, --list`",
		"`--stats`",
		"## Skills",
		"## atkins.yml",
		"### One idea, two spellings",
	} {
		assert.Contains(t, doc, want)
	}

	// A document written to a buffer is plain markdown with no escape codes.
	assert.NotContains(t, doc, "\x1b[")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(doc), "# atkins"))
}

// TestWriteHelpParentSkills checks the skill listing carries the skills
// of a parent folder, as a run from the same folder loads them.
func TestWriteHelpParentSkills(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ws := t.TempDir()
	proj := filepath.Join(ws, "proj")
	skillsDir := filepath.Join(ws, ".atkins", "skills")
	assert.NoError(t, os.MkdirAll(skillsDir, 0o755))
	assert.NoError(t, os.MkdirAll(proj, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(skillsDir, "wsonly.yml"), []byte("name: Workspace only\njobs:\n  default: pwd\n"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(proj, "atkins.yml"), []byte("jobs:\n  default: true\n"), 0o644))

	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})
	assert.NoError(t, os.Chdir(proj))

	skills, dirs := scanSkills()
	assert.Contains(t, dirs, skillsDir)

	var ids []string
	for _, skill := range skills {
		ids = append(ids, skill.ID())
	}
	assert.Contains(t, ids, "wsonly")

	var out bytes.Buffer
	assert.NoError(t, WriteHelp(&out))
	assert.Contains(t, out.String(), "wsonly")
}
