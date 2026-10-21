package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// writeFile writes a file under root, creating its folders.
func writeWhenFile(t *testing.T, root, name, contents string) string {
	t.Helper()

	path := filepath.Join(root, name)
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

// TestMatchWhen_Here checks here: true keeps the search to the starting
// folder, while the default search finds the file in a parent.
func TestMatchWhen_Here(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	assert.NoError(t, os.MkdirAll(sub, 0o755))
	compose := writeWhenFile(t, root, "compose.yml", "services: {}\n")

	match, ok := runner.MatchWhen(&model.PipelineWhen{Files: []string{"compose.yml"}}, sub)
	assert.True(t, ok)
	assert.Equal(t, root, match.Dir)
	assert.Equal(t, compose, match.File)

	_, ok = runner.MatchWhen(&model.PipelineWhen{Files: []string{"compose.yml"}, Here: true}, sub)
	assert.False(t, ok, "here: true ignores the parent's compose.yml")

	match, ok = runner.MatchWhen(&model.PipelineWhen{Files: []string{"compose.yml"}, Here: true}, root)
	assert.True(t, ok)
	assert.Equal(t, root, match.Dir)
}

// TestMatchWhen_Contains checks contains: requires the matched file to
// match the expression, skips folders, and looks through glob matches.
func TestMatchWhen_Contains(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pulse := filepath.Join(root, "pulse")
	task := filepath.Join(root, "task")
	writeWhenFile(t, pulse, "compose.yml", "services:\n  pulse:\n    build: .\n")
	writeWhenFile(t, task, "compose.yml", "services:\n  task:\n    image: task\n")

	build := &model.PipelineWhen{Files: []string{"compose.yml"}, Here: true, Contains: `(?m)^\s+build:`}

	match, ok := runner.MatchWhen(build, pulse)
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(pulse, "compose.yml"), match.File)

	_, ok = runner.MatchWhen(build, task)
	assert.False(t, ok, "task/compose.yml has no build:")

	// A folder never satisfies contains:, whatever its name.
	assert.NoError(t, os.MkdirAll(filepath.Join(task, "build.d"), 0o755))
	_, ok = runner.MatchWhen(&model.PipelineWhen{Files: []string{"build.d"}, Contains: "."}, task)
	assert.False(t, ok)

	// A glob tries every match, not only the first.
	writeWhenFile(t, root, "glob/a.yml", "image: a\n")
	writeWhenFile(t, root, "glob/b.yml", "build: .\n")
	match, ok = runner.MatchWhen(&model.PipelineWhen{Files: []string{"*.yml"}, Here: true, Contains: "build:"}, filepath.Join(root, "glob"))
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(root, "glob", "b.yml"), match.File)

	// An expression that does not compile never matches, and lints.
	bad := &model.PipelineWhen{Files: []string{"compose.yml"}, Contains: "("}
	_, ok = runner.MatchWhen(bad, pulse)
	assert.False(t, ok)
	assert.Error(t, runner.ValidateWhen(bad))
	assert.NoError(t, runner.ValidateWhen(build))
}

// TestApplyJobWhen checks a job's when: block marks it inactive where it
// does not match and leaves jobs without one alone.
func TestApplyJobWhen(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeWhenFile(t, root, "compose.yml", "services:\n  task:\n    image: task\n")

	build := &model.Job{When: &model.PipelineWhen{Files: []string{"compose.yml"}, Here: true, Contains: "build:"}}
	up := &model.Job{When: &model.PipelineWhen{Files: []string{"compose.yml"}, Here: true}}
	down := &model.Job{}
	pipeline := &model.Pipeline{ID: "compose", Jobs: map[string]*model.Job{"build": build, "up": up, "down": down}}

	runner.ApplyJobWhen([]*model.Pipeline{pipeline}, root)

	assert.True(t, build.Inactive)
	assert.False(t, up.Inactive)
	assert.Equal(t, filepath.Join(root, "compose.yml"), up.Matched)
	assert.False(t, down.Inactive)
}
