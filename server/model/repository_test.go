package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/server/model"
)

func TestRepositorySlug(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		expected string
	}{
		{"ssh scp syntax", "git@github.com:titpetric/atkins.git", "github.com/titpetric/atkins"},
		{"https", "https://github.com/titpetric/atkins.git", "github.com/titpetric/atkins"},
		{"https without suffix", "https://github.com/titpetric/atkins", "github.com/titpetric/atkins"},
		{"https with credentials", "https://token@github.com/titpetric/atkins", "github.com/titpetric/atkins"},
		{"ssh url", "ssh://git@github.com/titpetric/atkins.git", "github.com/titpetric/atkins"},
		{"trailing slash", "https://github.com/titpetric/atkins/", "github.com/titpetric/atkins"},
		{"mixed case", "https://GitHub.com/TitPetric/Atkins.git", "github.com/titpetric/atkins"},
		{"self hosted", "git@git.example.com:2222/team/tool.git", "git.example.com/2222/team/tool"},
		{"empty", "  ", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, model.RepositorySlug(test.remote))
		})
	}
}

func TestJobStatus(t *testing.T) {
	assert.True(t, model.ValidJobStatus(model.JobStatusPending))
	assert.False(t, model.ValidJobStatus("nonsense"))

	assert.False(t, model.TerminalJobStatus(model.JobStatusPending))
	assert.False(t, model.TerminalJobStatus(model.JobStatusRunning))
	assert.True(t, model.TerminalJobStatus(model.JobStatusPassed))
	assert.True(t, model.TerminalJobStatus(model.JobStatusTimeout))

	job := &model.Job{Status: model.JobStatusRunning}
	assert.False(t, job.IsTerminal())

	job.Status = model.JobStatusCancelled
	assert.True(t, job.IsTerminal())
}
