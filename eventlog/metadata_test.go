package eventlog

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
)

func TestExtractRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH URL",
			url:      "git@github.com:owner/repo.git",
			expected: "github.com/owner/repo",
		},
		{
			name:     "HTTPS URL",
			url:      "https://github.com/owner/repo.git",
			expected: "github.com/owner/repo",
		},
		{
			name:     "HTTP URL",
			url:      "http://github.com/owner/repo.git",
			expected: "github.com/owner/repo",
		},
		{
			name:     "No .git suffix",
			url:      "https://github.com/owner/repo",
			expected: "github.com/owner/repo",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:company/project.git",
			expected: "gitlab.com/company/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCaptureGitInfo(t *testing.T) {
	// This test runs in a git repository. CaptureGitInfo returns metadata
	info := CaptureGitInfo()

	// Nil outside a git repository
	if info != nil {
		assert.NotEmpty(t, info.Commit)
		assert.NotEmpty(t, info.Branch)
	}
}

func TestCaptureModulePath(t *testing.T) {
	// This test runs in a Go module. CaptureModulePath returns the path
	path := CaptureModulePath()

	// Should find the atkins module path
	assert.Contains(t, path, "atkins")
}
