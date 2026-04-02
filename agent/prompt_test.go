package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/agent"
)

func TestPromptMode_PromptChar(t *testing.T) {
	tests := []struct {
		mode     agent.PromptMode
		expected string
	}{
		{agent.PromptModeLanguage, ">"},
		{agent.PromptModeShell, "$"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.PromptChar())
		})
	}
}

func TestDetectPromptMode(t *testing.T) {
	tests := []struct {
		input    string
		expected agent.PromptMode
	}{
		{"", agent.PromptModeLanguage},
		{"hello", agent.PromptModeLanguage},
		{"run tests", agent.PromptModeLanguage},
		{"/list", agent.PromptModeLanguage},
		{"$", agent.PromptModeShell},
		{"$ ls", agent.PromptModeShell},
		{"$ ls -la", agent.PromptModeShell},
		{"$HOME", agent.PromptModeShell},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := agent.DetectPromptMode(tt.input)
			assert.Equal(t, tt.expected, result,
				"DetectPromptMode(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}
