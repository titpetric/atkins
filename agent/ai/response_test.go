package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{
			name:  "plain object",
			input: `{"message": "done"}`,
			want:  `{"message": "done"}`,
			ok:    true,
		},
		{
			name:  "surrounded by prose",
			input: "Sure, here you go:\n" + `{"cmds": ["atkins", "atkins release"]}` + "\nLet me know if you need more.",
			want:  `{"cmds": ["atkins", "atkins release"]}`,
			ok:    true,
		},
		{
			name:  "wrapped in a code fence",
			input: "```json\n" + `{"cmds": ["atkins release"]}` + "\n```",
			want:  `{"cmds": ["atkins release"]}`,
			ok:    true,
		},
		{
			name:  "brace inside a string value doesn't break balancing",
			input: `{"message": "added job { for build }"}`,
			want:  `{"message": "added job { for build }"}`,
			ok:    true,
		},
		{
			name:  "escaped quote inside a string value",
			input: `{"message": "wrote \"atkins.yml\""}`,
			want:  `{"message": "wrote \"atkins.yml\""}`,
			ok:    true,
		},
		{
			name:  "no object present",
			input: "no can do",
			ok:    false,
		},
		{
			name:  "unbalanced object",
			input: `{"cmds": ["atkins"]`,
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractJSON(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	resp, err := ParseResponse("Running that for you:\n" + `{"cmds": ["atkins release"]}`)
	require.NoError(t, err)
	assert.Equal(t, []string{"atkins release"}, resp.Cmds)
	assert.Empty(t, resp.Message)

	resp, err = ParseResponse(`{"message": "Added test:cover to atkins.yml"}`)
	require.NoError(t, err)
	assert.Equal(t, "Added test:cover to atkins.yml", resp.Message)

	_, err = ParseResponse("I can't help with that")
	assert.Error(t, err)

	_, err = ParseResponse(`{"cmds": "atkins release"}`)
	assert.Error(t, err)
}
