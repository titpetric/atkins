package treeview

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompactArgs(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		maxArgLen int
		expected  string
	}{
		{
			name:      "no args to compact",
			cmd:       "go build ./...",
			maxArgLen: 20,
			expected:  "go build ./...",
		},
		{
			name:      "short args unchanged",
			cmd:       "--foo=bar --baz=qux",
			maxArgLen: 10,
			expected:  "--foo=bar --baz=qux",
		},
		{
			name:      "long arg compacted",
			cmd:       "--ldflags=-X_main.version=1.0.0-beta.1-g12345678",
			maxArgLen: 10,
			expected:  "--ldflags=<...38 chars>",
		},
		{
			name:      "mixed args",
			cmd:       "go build --ldflags=-X_main.version=abcdefghij1234567890 -o ./bin/app",
			maxArgLen: 15,
			expected:  "go build --ldflags=<...36 chars> -o ./bin/app",
		},
		{
			name:      "multiple long args",
			cmd:       "--arg1=verylongvalue12345 --arg2=anotherlongvalue67890",
			maxArgLen: 10,
			expected:  "--arg1=<...18 chars> --arg2=<...21 chars>",
		},
		{
			name:      "disabled with zero maxArgLen",
			cmd:       "--foo=verylongvalue",
			maxArgLen: 0,
			expected:  "--foo=verylongvalue",
		},
		{
			name:      "single dash flag",
			cmd:       "-X=main.buildVersion=12345678901234567890",
			maxArgLen: 10,
			expected:  "-X=<...38 chars>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompactArgs(tt.cmd, tt.maxArgLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimmer_TrimToViewport(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		viewportWidth int
		prefixLen     int
		expected      string
	}{
		{
			name:          "no trimming needed",
			line:          "short line",
			viewportWidth: 80,
			prefixLen:     10,
			expected:      "short line",
		},
		{
			name:          "line trimmed",
			line:          "this is a very long line that exceeds the viewport",
			viewportWidth: 40,
			prefixLen:     10,
			expected:      "this is a very long line th...",
		},
		{
			name:          "zero viewport disables trimming",
			line:          "this is a very long line",
			viewportWidth: 0,
			prefixLen:     10,
			expected:      "this is a very long line",
		},
		{
			name:          "exact fit",
			line:          "exact",
			viewportWidth: 15,
			prefixLen:     10,
			expected:      "exact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmer := NewTrimmer()
			trimmer.SetViewportWidth(tt.viewportWidth)

			result := trimmer.TrimToViewport(tt.line, tt.prefixLen)

			// Strip ANSI reset code for comparison
			result = stripResetCode(result)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimmer_TrimLabel(t *testing.T) {
	trimmer := NewTrimmer()
	trimmer.SetViewportWidth(60)

	t.Run("compacts and trims", func(t *testing.T) {
		label := "run: go build --ldflags=-X_main.version=very-long-version-string-here"
		result := trimmer.TrimLabel(label, 15, 10)

		// Should have compacted the ldflags (no spaces in value, so it gets compacted)
		assert.Contains(t, result, "<...")
		// Should end with ... if trimmed to viewport
		assert.True(t, len(result) > 0)
	})

	t.Run("handles ANSI codes", func(t *testing.T) {
		label := "\033[1m\033[37mrun: some command\033[0m"
		result := trimmer.TrimToViewport(label, 10)

		// Should preserve the beginning of the label
		assert.Contains(t, result, "run:")
	})
}

func TestTrimmer_GetSetViewportWidth(t *testing.T) {
	trimmer := NewTrimmer()

	trimmer.SetViewportWidth(120)
	assert.Equal(t, 120, trimmer.GetViewportWidth())

	trimmer.SetViewportWidth(80)
	assert.Equal(t, 80, trimmer.GetViewportWidth())
}

// stripResetCode removes all ANSI reset codes added by trimWithANSI
func stripResetCode(s string) string {
	const resetCode = "\033[0m"
	result := s
	for {
		idx := len(result) - len(resetCode)
		if idx >= 0 && result[idx:] == resetCode {
			result = result[:idx]
		} else {
			break
		}
	}
	// Also strip reset codes that appear before "..."
	result = strings.ReplaceAll(result, resetCode, "")
	return result
}
