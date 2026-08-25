package ai

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAtkinsCmds(t *testing.T) {
	tests := []struct {
		name    string
		cmds    []string
		wantErr bool
		want    [][]string
	}{
		{
			name: "single atkins command",
			cmds: []string{"atkins release"},
			want: [][]string{{"atkins", "release"}},
		},
		{
			name: "bare atkins",
			cmds: []string{"atkins"},
			want: [][]string{{"atkins"}},
		},
		{
			name: "multiple atkins commands",
			cmds: []string{"atkins", "atkins release"},
			want: [][]string{{"atkins"}, {"atkins", "release"}},
		},
		{
			name: "quoted argument stays one argument",
			cmds: []string{`atkins -x "run the tests"`},
			want: [][]string{{"atkins", "-x", "run the tests"}},
		},
		{
			name: "a path spelling of atkins is accepted",
			cmds: []string{"./atkins release", "/usr/local/bin/atkins -l"},
			want: [][]string{{"./atkins", "release"}, {"/usr/local/bin/atkins", "-l"}},
		},
		{
			name:    "another binary is rejected",
			cmds:    []string{"atkins release", "rm -rf /"},
			wantErr: true,
		},
		{
			name:    "a name atkins is a prefix of is rejected",
			cmds:    []string{"atkinsd serve"},
			wantErr: true,
		},
		{
			name:    "empty command rejected",
			cmds:    []string{""},
			wantErr: true,
		},
		{
			name:    "unterminated quote rejected",
			cmds:    []string{`atkins -x "run the tests`},
			wantErr: true,
		},
		{
			name: "shell metacharacters in an atkins command are inert",
			// There is no shell in the execution path (argv only), so a
			// value like this is never interpreted: atkins is handed the
			// argument "release;" followed by "rm", "-rf", "/".
			cmds: []string{"atkins release; rm -rf /"},
			want: [][]string{{"atkins", "release;", "rm", "-rf", "/"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAtkinsCmds(tt.cmds)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		want    []string
		wantErr bool
	}{
		{name: "plain words", cmd: "atkins go:test", want: []string{"atkins", "go:test"}},
		{name: "repeated whitespace", cmd: "  atkins   -l  ", want: []string{"atkins", "-l"}},
		{name: "double quotes", cmd: `atkins -x "run the tests"`, want: []string{"atkins", "-x", "run the tests"}},
		{name: "single quotes", cmd: `atkins -x 'run the tests'`, want: []string{"atkins", "-x", "run the tests"}},
		{name: "empty quoted argument", cmd: `atkins -x ""`, want: []string{"atkins", "-x", ""}},
		{name: "quotes inside a word", cmd: `atkins -x"a b"`, want: []string{"atkins", "-xa b"}},
		{name: "escaped space", cmd: `atkins a\ b`, want: []string{"atkins", "a b"}},
		{name: "backslash inside single quotes is literal", cmd: `atkins 'a\b'`, want: []string{"atkins", `a\b`}},
		{name: "empty string", cmd: "", want: nil},
		{name: "unterminated double quote", cmd: `atkins "a`, wantErr: true},
		{name: "trailing backslash", cmd: `atkins a\`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitArgs(tt.cmd)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormat(t *testing.T) {
	assert.Equal(t, "atkins -l", Format([]string{"atkins", "-l"}))
	assert.Equal(t, `atkins -x "run the tests"`, Format([]string{"atkins", "-x", "run the tests"}))
}

func TestRun(t *testing.T) {
	var out bytes.Buffer

	// argv[0] is dropped in favour of the binary the caller resolved, so
	// the suggestion decides the arguments and nothing else.
	err := run(context.Background(), "/bin/echo", t.TempDir(), [][]string{
		{"atkins", "first"},
		{"atkins", "second"},
	}, &out, &out)

	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", out.String())
}

func TestRun_StopsAtFirstFailure(t *testing.T) {
	var out bytes.Buffer

	err := run(context.Background(), "/bin/false", t.TempDir(), [][]string{
		{"atkins", "one"},
		{"atkins", "two"},
	}, &out, &out)

	assert.Error(t, err)
	assert.Empty(t, out.String())
}

func TestRun_UsesWorkDir(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer

	err := run(context.Background(), "/bin/pwd", dir, [][]string{{"atkins"}}, &out, &out)

	require.NoError(t, err)
	assert.Contains(t, out.String(), dir)
}
