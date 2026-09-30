package api

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
)

func TestEncodeParams(t *testing.T) {
	encoded, err := encodeParams(nil)
	assert.NoError(t, err)
	assert.Equal(t, "{}", encoded)

	encoded, err = encodeParams(map[string]any{"tag": "v1.2.3"})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"tag":"v1.2.3"}`, encoded)
}
