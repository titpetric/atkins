package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanWorkingDirectory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"dot", ".", ""},
		{"root", "/", ""},
		{"relative", "docs", "docs"},
		{"nested", "server/api", "server/api"},
		{"leading slash is dropped", "/server/api", "server/api"},
		{"redundant segments", "server/./api/", "server/api"},
		{"windows separators", `server\api`, "server/api"},
		{"escape is refused", "../secrets", ""},
		{"escape through a segment is refused", "server/../../secrets", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, cleanWorkingDirectory(test.input))
		})
	}
}

func TestEncodeParams(t *testing.T) {
	encoded, err := encodeParams(nil)
	require.NoError(t, err)
	assert.Equal(t, "{}", encoded)

	encoded, err = encodeParams(map[string]any{"tag": "v1.2.3"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"tag":"v1.2.3"}`, encoded)
}
