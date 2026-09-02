package helpdoc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/helpdoc"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestOptions checks the zero value renders and that the fields reach
// the document.
func TestOptions(t *testing.T) {
	opts := &helpdoc.Options{}
	assert.Empty(t, opts.Name)
	assert.Empty(t, opts.Skills)
	assert.False(t, opts.Color)

	opts = &helpdoc.Options{
		Name:      "atkins",
		Version:   "Version dev.",
		SkillDirs: []string{"/tmp/.atkins/skills"},
		Skills: []*runner.Skill{{
			Pipeline: &model.Pipeline{ID: "go"},
			Path:     "/tmp/.atkins/skills/go.yml",
			Active:   true,
		}},
	}

	assert.Equal(t, "atkins", opts.Name)
	assert.Len(t, opts.Skills, 1)
	assert.Equal(t, "go", opts.Skills[0].ID())
}
