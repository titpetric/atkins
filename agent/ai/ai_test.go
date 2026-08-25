package ai_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/agent/ai"
	"github.com/titpetric/atkins/model"
)

func testPipelines() []*model.Pipeline {
	return []*model.Pipeline{
		{
			Name: "My Project",
			Jobs: map[string]*model.Job{
				"default": {Name: "default", Desc: "Run everything"},
			},
		},
	}
}

func TestBuildPrompt(t *testing.T) {
	dir := t.TempDir()
	yamlContent := "jobs:\n  default:\n    cmd: echo hi\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "atkins.yml"), []byte(yamlContent), 0o644))

	prompt := ai.BuildPrompt("deploy the release", testPipelines(), dir)

	assert.Contains(t, prompt, "The user writes: deploy the release")
	assert.Contains(t, prompt, "The pipelines available are:")
	assert.Contains(t, prompt, "default")
	assert.Contains(t, prompt, "Atkins pipeline: "+yamlContent)
	assert.Contains(t, prompt, `{"cmds": ["atkins", "atkins release"]}`)
	assert.Contains(t, prompt, `{"cmds": ["atkins -l"]}`)
}

func TestBuildPrompt_NoConfigFile(t *testing.T) {
	dir := t.TempDir()

	prompt := ai.BuildPrompt("list jobs", testPipelines(), dir)

	assert.Contains(t, prompt, "(no atkins.yml found)")
}

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
			name:    "non-atkins command rejected",
			cmds:    []string{"atkins release", "rm -rf /"},
			wantErr: true,
		},
		{
			name:    "empty command rejected",
			cmds:    []string{""},
			wantErr: true,
		},
		{
			name: "shell metacharacters in an atkins command are inert",
			// There's no shell in the execution path (argv only), so a
			// value like this is never interpreted - it's passed to
			// atkins literally as the single argument "release;" followed
			// by "rm", "-rf", "/", none of which is a shell that atkins
			// itself would re-interpret.
			cmds: []string{"atkins release; rm -rf /"},
			want: [][]string{{"atkins", "release;", "rm", "-rf", "/"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ai.ValidateAtkinsCmds(tt.cmds)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrompt_LiteralPercentSNotAFormatVerb(t *testing.T) {
	// BuildPrompt must not run its static instructional text through
	// fmt.Sprintf - the example message text contains a literal "%s" that
	// would otherwise need a matching argument.
	prompt := ai.BuildPrompt("x", nil, t.TempDir())
	assert.True(t, strings.Contains(prompt, "Added new %s target to atkins.yml"))
}
