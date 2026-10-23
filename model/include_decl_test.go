package model_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestIncludeDecl_UnmarshalYAML checks include: takes a string or a list and
// refuses a mapping.
func TestIncludeDecl_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	var one model.IncludeDecl
	assert.NoError(t, yaml.Unmarshal([]byte(`vars.yml`), &one))
	assert.Equal(t, []string{"vars.yml"}, one.Files)

	var list model.IncludeDecl
	assert.NoError(t, yaml.Unmarshal([]byte(`[a.yml, b.yml]`), &list))
	assert.Equal(t, []string{"a.yml", "b.yml"}, list.Files)

	var bad model.IncludeDecl
	assert.Error(t, yaml.Unmarshal([]byte(`{a: b}`), &bad))
}
