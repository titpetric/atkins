package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtefactPath(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"plain file", "scan.json", "scan.json"},
		{"nested", "reports/scan.json", "reports/scan.json"},
		{"leading dot slash", "./scan.json", "scan.json"},
		{"doubled separators", "reports//scan.json", "reports/scan.json"},
		{"windows separators", `reports\scan.json`, "reports/scan.json"},
		{"whitespace", "  scan.json  ", "scan.json"},

		// Everything below would address a file the job never
		// produced, and is refused rather than repaired.
		{"traversal", "../../etc/passwd", ""},
		{"traversal inside", "reports/../../escape", ""},
		{"absolute", "/etc/passwd", ""},
		{"windows absolute", `\etc\passwd`, ""},
		{"bare traversal", "..", ""},
		{"empty", "", ""},
		{"only separators", "///", ""},
		{"newline in the name", "scan\n.json", ""},
		{"too long", strings.Repeat("a", MaxArtefactPathLength+1), ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, ArtefactPath(test.value))
		})
	}
}

func TestMatchArtefactPath(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		value    string
		expected bool
	}{
		{"exact", "scan.json", "scan.json", true},
		{"extension in a directory", "reports/*.json", "reports/scan.json", true},
		{"star does not cross a separator", "*.json", "reports/scan.json", false},
		{"double star crosses", "**/*.json", "a/b/scan.json", true},
		{"double star matches nothing", "**/scan.json", "scan.json", true},
		{"everything", "**", "a/b/c.txt", true},
		{"prefix", "coverage*", "coverage.out", true},
		{"no match", "*.json", "scan.xml", false},

		// Filenames are case-sensitive on the systems agents run on,
		// so matching is too.
		{"case matters", "*.json", "SCAN.JSON", false},

		{"empty pattern", "", "scan.json", false},
		{"empty value", "*.json", "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, MatchArtefactPath(test.pattern, test.value))
		})
	}
}

func TestArtefactPatterns(t *testing.T) {
	assert.Nil(t, ArtefactPatterns(""))
	assert.Nil(t, ArtefactPatterns("  ,  "))
	assert.Equal(t, []string{"*.json", "coverage/*"}, ArtefactPatterns(" *.json , coverage/* "))

	// A pattern that could leave the checkout is dropped, and the rest
	// of the list still stands.
	assert.Equal(t, []string{"*.json"}, ArtefactPatterns("*.json,../../etc/*,/etc/*"))
}

func TestJoinArtefactPatterns(t *testing.T) {
	assert.Equal(t, "", JoinArtefactPatterns(nil))
	assert.Equal(t, "*.json,dist/*", JoinArtefactPatterns([]string{"*.json", "../escape", "./dist/*"}))
}

func TestArtefactContentType(t *testing.T) {
	assert.Equal(t, "application/json", ArtefactContentType("application/json"))
	assert.Equal(t, DefaultContentType, ArtefactContentType(""))
	assert.Equal(t, DefaultContentType, ArtefactContentType(strings.Repeat("x", 200)))

	// The value is echoed back as a response header, so a smuggled
	// newline must not survive.
	assert.Equal(t, DefaultContentType, ArtefactContentType("text/html\r\nSet-Cookie: a=b"))
}
