package runner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestSkill_ID checks ID reads the namespace off the pipeline and
// answers for a zero value rather than panicking.
func TestSkill_ID(t *testing.T) {
	var missing *runner.Skill
	assert.Empty(t, missing.ID())
	assert.Empty(t, (&runner.Skill{}).ID())
	assert.Equal(t, "go", (&runner.Skill{Pipeline: &model.Pipeline{ID: "go"}}).ID())
}

// TestSkill checks the fields a skill carries.
func TestSkill(t *testing.T) {
	skill := &runner.Skill{
		Pipeline: &model.Pipeline{ID: "go", Name: "Go build and test"},
		Path:     "/home/user/.atkins/skills/go.yml",
		DocPath:  "/home/user/.atkins/skills/go.md",
		Doc:      "`atkins go:fmt` formats the sources.",
		Dir:      "/repo",
		Active:   true,
	}

	assert.Equal(t, "go", skill.ID())
	assert.True(t, skill.Active)
	assert.Equal(t, "/repo", skill.Dir)
}
