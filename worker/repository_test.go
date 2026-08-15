package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloneURL(t *testing.T) {
	tests := []struct {
		name        string
		remote      string
		slug        string
		preferHTTPS bool
		expected    string
	}{
		{
			name:     "left alone when https is not preferred",
			remote:   "git@github.com:titpetric/atkins.git",
			slug:     "github.com/titpetric/atkins",
			expected: "git@github.com:titpetric/atkins.git",
		},
		{
			name:        "scp syntax rewritten",
			remote:      "git@github.com:titpetric/atkins.git",
			slug:        "github.com/titpetric/atkins",
			preferHTTPS: true,
			expected:    "https://github.com/titpetric/atkins.git",
		},
		{
			name:        "ssh url rewritten",
			remote:      "ssh://git@github.com/titpetric/atkins.git",
			slug:        "github.com/titpetric/atkins",
			preferHTTPS: true,
			expected:    "https://github.com/titpetric/atkins.git",
		},
		{
			name:        "https left alone",
			remote:      "https://github.com/titpetric/atkins.git",
			slug:        "github.com/titpetric/atkins",
			preferHTTPS: true,
			expected:    "https://github.com/titpetric/atkins.git",
		},
		{
			name:        "a local path is not a host, so it is left alone",
			remote:      "/srv/git/demo.git",
			slug:        "/srv/git/demo",
			preferHTTPS: true,
			expected:    "/srv/git/demo.git",
		},
		{
			name:        "a slug without a dotted host is left alone",
			remote:      "git@internal:team/tool.git",
			slug:        "internal/team/tool",
			preferHTTPS: true,
			expected:    "git@internal:team/tool.git",
		},
		{
			name:        "no slug means nothing to rewrite to",
			remote:      "git@github.com:titpetric/atkins.git",
			slug:        "",
			preferHTTPS: true,
			expected:    "git@github.com:titpetric/atkins.git",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, CloneURL(test.remote, test.slug, test.preferHTTPS))
		})
	}
}

func TestIsSSHRemote(t *testing.T) {
	assert.True(t, isSSHRemote("git@github.com:titpetric/atkins.git"))
	assert.True(t, isSSHRemote("ssh://git@github.com/titpetric/atkins.git"))
	assert.False(t, isSSHRemote("https://github.com/titpetric/atkins.git"))
	assert.False(t, isSSHRemote("/srv/git/demo.git"))
}

func TestCachePath(t *testing.T) {
	assert.Equal(t, "github.com/titpetric/atkins.git", cachePath("github.com/titpetric/atkins"))

	// A slug that tries to climb out is flattened into the cache root
	// rather than resolving above it.
	assert.Equal(t, "etc/passwd.git", cachePath("../../etc/passwd"))
	assert.Equal(t, "srv/git/demo.git", cachePath("/srv/git/demo"))
}

func TestCleanWorkingDirectory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{".", ""},
		{"/", ""},
		{"docs", "docs"},
		{"server/api", "server/api"},
		{"/server/api", "server/api"},
		{"server/./api/", "server/api"},
		{`server\api`, "server/api"},
		// An agent must not run a shell command outside its checkout,
		// whatever the server stored.
		{"../secrets", ""},
		{"server/../../secrets", ""},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.expected, CleanWorkingDirectory(test.input))
		})
	}
}
