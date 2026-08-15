package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeParams(t *testing.T) {
	encoded, err := encodeParams(nil)
	require.NoError(t, err)
	assert.Equal(t, "{}", encoded)

	encoded, err = encodeParams(map[string]any{"tag": "v1.2.3"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"tag":"v1.2.3"}`, encoded)
}
