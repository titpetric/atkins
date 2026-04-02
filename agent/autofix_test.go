package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/agent"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

func createTestPipelinesWithFix() []*model.Pipeline {
	return []*model.Pipeline{
		{
			ID: "go",
			Jobs: map[string]*model.Job{
				"test":  {Name: "test", Desc: "Run tests"},
				"build": {Name: "build", Desc: "Build the project"},
				"fix":   {Name: "fix", Desc: "Fix linting issues"},
			},
		},
		{
			ID: "docker",
			Jobs: map[string]*model.Job{
				"up":   {Name: "up", Desc: "Start containers"},
				"down": {Name: "down", Desc: "Stop containers"},
				// No fix job
			},
		},
	}
}

func TestAutoFixer_NewAutoFixer(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)

	fixer := agent.NewAutoFixer(resolver, pipelines)
	assert.NotNil(t, fixer)
}

func TestAutoFixer_CanFix_WithFixTask(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)
	fixer := agent.NewAutoFixer(resolver, pipelines)

	// go:test has go:fix
	goTest, err := resolver.Resolve("go:test")
	assert.NoError(t, err)

	canFix := fixer.CanFix(goTest)
	assert.True(t, canFix)
}

func TestAutoFixer_CanFix_WithoutFixTask(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)
	fixer := agent.NewAutoFixer(resolver, pipelines)

	// docker:up has no docker:fix
	dockerUp, err := resolver.Resolve("docker:up")
	assert.NoError(t, err)

	canFix := fixer.CanFix(dockerUp)
	assert.False(t, canFix)
}

func TestAutoFixer_CanFix_NilTask(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)
	fixer := agent.NewAutoFixer(resolver, pipelines)

	assert.False(t, fixer.CanFix(nil))
}

func TestAutoFixer_GetFixTask(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)
	fixer := agent.NewAutoFixer(resolver, pipelines)

	goTest, err := resolver.Resolve("go:test")
	assert.NoError(t, err)

	fixTask, err := fixer.GetFixTask(goTest)
	assert.NoError(t, err)
	assert.NotNil(t, fixTask)
	assert.Equal(t, "go:fix", fixTask.Name)
}

func TestAutoFixer_GetFixTask_NotFound(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)
	fixer := agent.NewAutoFixer(resolver, pipelines)

	dockerUp, err := resolver.Resolve("docker:up")
	assert.NoError(t, err)

	fixTask, err := fixer.GetFixTask(dockerUp)
	assert.Error(t, err)
	assert.Nil(t, fixTask)
}

func TestAutoFixer_GetFixTask_NilTask(t *testing.T) {
	pipelines := createTestPipelinesWithFix()
	resolver := runner.NewTaskResolver(pipelines)
	fixer := agent.NewAutoFixer(resolver, pipelines)

	fixTask, err := fixer.GetFixTask(nil)
	assert.NoError(t, err)
	assert.Nil(t, fixTask)
}

func TestDefaultAutoFixConfig(t *testing.T) {
	cfg := agent.DefaultAutoFixConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 1, cfg.MaxRetries)
	assert.Equal(t, "fix", cfg.FixTaskSuffix)
}

func TestAutoFixConfig_Fields(t *testing.T) {
	cfg := &agent.AutoFixConfig{
		Enabled:       false,
		MaxRetries:    3,
		FixTaskSuffix: "autofix",
	}

	assert.False(t, cfg.Enabled)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, "autofix", cfg.FixTaskSuffix)
}
