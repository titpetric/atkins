package runner_test

import (
	"errors"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

func TestFuzzyMatchError(t *testing.T) {
	t.Parallel()

	pipeline := &model.Pipeline{Name: "main", Jobs: map[string]*model.Job{
		"unit:test":        {Name: "unit:test"},
		"integration:test": {Name: "integration:test"},
	}}

	_, err := runner.NewTaskResolver([]*model.Pipeline{pipeline}).Resolve("test")

	var fuzzy *runner.FuzzyMatchError
	assert.True(t, errors.As(err, &fuzzy), "two suffix matches are ambiguous")
	assert.Equal(t, 2, len(fuzzy.Matches))
}

func TestFuzzyMatchError_Error(t *testing.T) {
	t.Parallel()

	err := &runner.FuzzyMatchError{}
	assert.Contains(t, err.Error(), "use -l to see all jobs")
}
