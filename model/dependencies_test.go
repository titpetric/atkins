package model_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestDependencies_UnmarshalYAML checks depends_on takes a string or a list.
func TestDependencies_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	var one model.Dependencies
	assert.NoError(t, yaml.Unmarshal([]byte(`build`), &one))
	assert.Equal(t, model.Dependencies{"build"}, one)

	var list model.Dependencies
	assert.NoError(t, yaml.Unmarshal([]byte(`[build, test]`), &list))
	assert.Equal(t, model.Dependencies{"build", "test"}, list)

	var bad model.Dependencies
	assert.Error(t, yaml.Unmarshal([]byte(`{a: b}`), &bad))
}
