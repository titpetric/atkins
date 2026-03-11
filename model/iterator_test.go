package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestIterators_UnmarshalYAML_SingleString(t *testing.T) {
	input := `for: item in items`

	var step struct {
		For Iterators `yaml:"for"`
	}
	err := yaml.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	assert.Len(t, step.For, 1)
	assert.Equal(t, Iterator("item in items"), step.For[0])
	assert.False(t, step.For.IsEmpty())
}

func TestIterators_UnmarshalYAML_List(t *testing.T) {
	input := `for:
  - goos in build.goos
  - goarch in build.goarch`

	var step struct {
		For Iterators `yaml:"for"`
	}
	err := yaml.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	assert.Len(t, step.For, 2)
	assert.Equal(t, Iterator("goos in build.goos"), step.For[0])
	assert.Equal(t, Iterator("goarch in build.goarch"), step.For[1])
}

func TestIterators_UnmarshalYAML_InlineList(t *testing.T) {
	input := `for: ["a in as", "b in bs", "c in cs"]`

	var step struct {
		For Iterators `yaml:"for"`
	}
	err := yaml.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	assert.Len(t, step.For, 3)
	assert.Equal(t, Iterator("a in as"), step.For[0])
	assert.Equal(t, Iterator("b in bs"), step.For[1])
	assert.Equal(t, Iterator("c in cs"), step.For[2])
}

func TestIterators_IsEmpty(t *testing.T) {
	var empty Iterators
	assert.True(t, empty.IsEmpty())

	notEmpty := Iterators{"item in items"}
	assert.False(t, notEmpty.IsEmpty())
}

func TestConditionals_UnmarshalYAML_SingleString(t *testing.T) {
	input := `if: enabled == true`

	var step struct {
		If Conditionals `yaml:"if"`
	}
	err := yaml.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	assert.Len(t, step.If, 1)
	assert.Equal(t, Condition("enabled == true"), step.If[0])
	assert.False(t, step.If.IsEmpty())
}

func TestConditionals_UnmarshalYAML_List(t *testing.T) {
	input := `if:
  - enabled == true
  - count > 0`

	var step struct {
		If Conditionals `yaml:"if"`
	}
	err := yaml.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	assert.Len(t, step.If, 2)
	assert.Equal(t, Condition("enabled == true"), step.If[0])
	assert.Equal(t, Condition("count > 0"), step.If[1])
}

func TestConditionals_UnmarshalYAML_InlineList(t *testing.T) {
	input := `if: ["a == 1", "b == 2"]`

	var step struct {
		If Conditionals `yaml:"if"`
	}
	err := yaml.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	assert.Len(t, step.If, 2)
	assert.Equal(t, Condition("a == 1"), step.If[0])
	assert.Equal(t, Condition("b == 2"), step.If[1])
}

func TestConditionals_IsEmpty(t *testing.T) {
	var empty Conditionals
	assert.True(t, empty.IsEmpty())

	notEmpty := Conditionals{"enabled == true"}
	assert.False(t, notEmpty.IsEmpty())
}
