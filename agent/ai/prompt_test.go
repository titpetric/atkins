package ai_test

import (
	"os"
	"path/filepath"
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
	prompt := ai.BuildPrompt("list jobs", testPipelines(), t.TempDir())

	assert.Contains(t, prompt, "(no atkins.yml found)")
}

func TestBuildPrompt_LiteralPercentSNotAFormatVerb(t *testing.T) {
	// The instructions carry a literal "%s" as part of an example message,
	// so the prompt is assembled by concatenation and never by fmt.Sprintf.
	prompt := ai.BuildPrompt("x", nil, t.TempDir())

	assert.Contains(t, prompt, "Added new %s target to atkins.yml")
}
