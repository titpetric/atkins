package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateStepID(t *testing.T) {
	tests := []struct {
		name      string
		jobName   string
		stepIndex int
		expected  string
	}{
		{
			name:      "simple job and step",
			jobName:   "test-job",
			stepIndex: 0,
			expected:  "jobs.test-job.steps.0",
		},
		{
			name:      "job with colon",
			jobName:   "docker:up",
			stepIndex: 5,
			expected:  "jobs.docker:up.steps.5",
		},
		{
			name:      "empty job name",
			jobName:   "",
			stepIndex: 0,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateStepID(tt.jobName, tt.stepIndex)
			assert.Equal(t, tt.expected, result)
		})
	}
}
