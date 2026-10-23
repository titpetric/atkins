package runner_test

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

func TestListPipelines(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", runner.ListPipelines(nil))

	main := &model.Pipeline{Name: "Project", Jobs: map[string]*model.Job{
		"default": {Name: "default", Desc: "Run everything", DependsOn: []string{"build"}},
		"build":   {Name: "build", Aliases: []string{"b"}},
	}}
	skill := &model.Pipeline{ID: "go", Name: "Go", Jobs: map[string]*model.Job{
		"default": {Name: "default"},
		"test":    {Name: "test", Desc: "Run the tests"},
	}}

	out := runner.ListPipelines([]*model.Pipeline{skill, main})

	sections := strings.Split(out, "\n\n")
	assert.Contains(t, sections[0], "Project", "the project pipeline comes first")
	assert.Contains(t, out, "Run everything")
	assert.Contains(t, out, "depends_on:")
	assert.Contains(t, out, "aliases:")
	assert.Contains(t, out, "Aliases", "a skill with a default job adds an alias section")
	assert.Contains(t, out, "go:test")
	assert.Contains(t, out, "Run the tests")
	assert.True(t, strings.HasSuffix(out, "\n"))
}
