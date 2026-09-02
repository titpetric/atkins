package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDiffLines checks a file splits into lines that keep their newline,
// with no phantom line for the trailing one, which is what keeps the
// vendoring line counts off by nothing.
func TestDiffLines(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{name: "empty", data: "", want: nil},
		{name: "one line with a newline", data: "a\n", want: []string{"a\n"}},
		{name: "one line without a newline", data: "a", want: []string{"a"}},
		{name: "two lines", data: "a\nb\n", want: []string{"a\n", "b\n"}},
		{name: "trailing blank line", data: "a\n\n", want: []string{"a\n", "\n"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, diffLines([]byte(test.data)))
		})
	}
}
