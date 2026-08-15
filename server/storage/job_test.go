package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentAcceptsJob(t *testing.T) {
	tests := []struct {
		name        string
		agentLabels []string
		jobLabels   string
		expected    bool
	}{
		{"unlabelled job runs anywhere", nil, "", true},
		{"unlabelled job on labelled agent", []string{"linux"}, "  ", true},
		{"exact match", []string{"linux"}, "linux", true},
		{"agent offers more than required", []string{"linux", "arm64", "docker"}, "linux,arm64", true},
		{"agent missing one label", []string{"linux", "amd64"}, "linux,arm64", false},
		{"unlabelled agent, labelled job", nil, "linux", false},
		{"whitespace tolerated", []string{"linux", "arm64"}, " linux , arm64 ", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, agentAcceptsJob(test.agentLabels, test.jobLabels))
		})
	}
}

func TestSplitLabels(t *testing.T) {
	assert.Nil(t, splitLabels(""))
	assert.Nil(t, splitLabels("   "))
	assert.Equal(t, []string{"linux", "arm64"}, splitLabels("linux,arm64"))
	assert.Equal(t, []string{"linux", "arm64"}, splitLabels(" linux , , arm64 "))
}
