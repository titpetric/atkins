package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpRequested covers the command lines that ask for the help
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
		{name: "after a command", args: []string{"server", "--help"}, want: true},
		{name: "after a job name", args: []string{"go:fmt", "--help"}, want: true},
		{name: "after a boolean flag", args: []string{"--final", "-h"}, want: true},
		{name: "value of a string flag", args: []string{"-x", "--help"}, want: false},
		{name: "value of a long string flag", args: []string{"--exec", "--help"}, want: false},
		{name: "string flag with an inline value", args: []string{"--exec=run", "--help"}, want: true},
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

// TestHelpFlags checks the merged set carries a flag from each of the
// three commands, which is what tells a value flag from a boolean one.
func TestHelpFlags(t *testing.T) {
	flags := HelpFlags()

	require.NotNil(t, flags.Lookup("exec"), "pipeline flag")
	require.NotNil(t, flags.Lookup("signing-key"), "server flag")
	require.NotNil(t, flags.Lookup("agent-id"), "worker flag")

	assert.Equal(t, "bool", flags.Lookup("final").Value.Type())
	assert.Equal(t, "string", flags.Lookup("exec").Value.Type())
}

// TestWriteHelp renders the document the binary prints and checks it
// carries the usage line, every command, the flags and the schema.
func TestWriteHelp(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, WriteHelp(&out))

	doc := out.String()

	for _, want := range []string{
		"# atkins",
		"Usage: atkins [flags] [job...]",
		"## Commands",
		"`run`",
		"`server`",
		"`worker`",
		"## Flags",
		"`-l, --list`",
		"## Flags for server",
		"`--signing-key`",
		"## Flags for worker",
		"`--agent-id`",
		"## Skills",
		"## atkins.yml",
		"### One idea, several spellings",
	} {
		assert.Contains(t, doc, want)
	}

	// A document written to a buffer is markdown, not a screenshot.
	assert.NotContains(t, doc, "\x1b[")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(doc), "# atkins"))
}
