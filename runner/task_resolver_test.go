package runner_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// resolverPipelines are a project pipeline and a skill, both with a build job.
func resolverPipelines() (main, skill *model.Pipeline) {
	main = &model.Pipeline{Name: "main", Jobs: map[string]*model.Job{
		"build":       {Name: "build", Aliases: []string{"b"}},
		"integration": {Name: "integration"},
	}}
	skill = &model.Pipeline{ID: "go", Name: "go", Jobs: map[string]*model.Job{
		"build": {Name: "build"},
		"test":  {Name: "test"},
	}}
	return main, skill
}

func TestTaskResolver(t *testing.T) {
	t.Parallel()

	main, skill := resolverPipelines()
	var resolver *runner.TaskResolver = runner.NewTaskResolver([]*model.Pipeline{main, skill})

	resolved, err := resolver.Resolve("go:test")
	assert.NoError(t, err)
	assert.Equal(t, skill, resolved.Pipeline)
	assert.Equal(t, "go:test", resolved.Name)
}

func TestNewTaskResolver(t *testing.T) {
	t.Parallel()

	main, skill := resolverPipelines()
	resolved, err := runner.NewTaskResolver([]*model.Pipeline{main, skill}).Resolve("build")
	assert.NoError(t, err)
	assert.Equal(t, main, resolved.Pipeline, "an exact name resolves to the project's own job")
}

func TestNewSkillResolver(t *testing.T) {
	t.Parallel()

	_, skill := resolverPipelines()
	resolver := runner.NewSkillResolver(skill)

	resolved, err := resolver.Resolve("test")
	assert.NoError(t, err)
	assert.Equal(t, skill, resolved.Pipeline)

	_, err = resolver.Resolve("integration")
	assert.Error(t, err, "a skill resolver sees its own pipeline alone")
}

func TestTaskResolver_Resolve(t *testing.T) {
	t.Parallel()

	main, skill := resolverPipelines()
	resolver := runner.NewTaskResolver([]*model.Pipeline{main, skill})

	resolved, err := resolver.Resolve(":build")
	assert.NoError(t, err)
	assert.Equal(t, main, resolved.Pipeline)

	_, err = resolver.Resolve(":b")
	assert.Error(t, err, "a leading colon takes exact names, no aliases")
}

func TestTaskResolver_ResolveWithFallback(t *testing.T) {
	t.Parallel()

	main, skill := resolverPipelines()
	global := runner.NewTaskResolver([]*model.Pipeline{main, skill})
	local := runner.NewSkillResolver(skill)

	resolved, err := local.ResolveWithFallback("build", global)
	assert.NoError(t, err)
	assert.Equal(t, skill, resolved.Pipeline, "the skill's own job comes first")

	resolved, err = local.ResolveWithFallback("integration", global)
	assert.NoError(t, err)
	assert.Equal(t, main, resolved.Pipeline, "a name the skill lacks falls back")

	resolved, err = local.ResolveWithFallback(":build", global)
	assert.NoError(t, err)
	assert.Equal(t, main, resolved.Pipeline, "a leading colon skips the skill")
}

func TestTaskResolver_ResolveName(t *testing.T) {
	t.Parallel()

	main, skill := resolverPipelines()
	resolver := runner.NewTaskResolver([]*model.Pipeline{main, skill})

	resolved, err := resolver.ResolveName("b", false)
	assert.NoError(t, err)
	assert.Equal(t, "build", resolved.Name, "an alias resolves to its job")

	resolved, err = resolver.ResolveName("gration", false)
	assert.NoError(t, err)
	assert.Equal(t, "integration", resolved.Name, "a suffix resolves fuzzily")

	_, err = resolver.ResolveName("b", true)
	assert.Error(t, err, "strict resolution takes exact names alone")

	_, err = resolver.ResolveName("missing", false)
	assert.ErrorContains(t, err, "not found")
}
