package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		value    string
		expected int64
		invalid  bool
	}{
		{value: "0", expected: 0},
		{value: "1024", expected: 1024},
		{value: "64B", expected: 64},
		{value: "512KB", expected: 512 * 1024},
		{value: "32MB", expected: 32 * 1024 * 1024},
		{value: "1GB", expected: 1024 * 1024 * 1024},
		{value: " 8 mb ", expected: 8 * 1024 * 1024},

		{value: "", invalid: true},
		{value: "a handful", invalid: true},
		{value: "-1", invalid: true},
		{value: "12TB", invalid: true},
		// The multiplication must not be allowed to wrap into a
		// negative limit, which would refuse every upload.
		{value: "9223372036854775807GB", invalid: true},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			parsed, err := ParseBytes(test.value)
			if test.invalid {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, parsed)
		})
	}
}

func TestArtefactSettingsAreRegistered(t *testing.T) {
	for _, name := range []string{
		SettingArtefactMaxSize,
		SettingArtefactMaxCount,
		SettingArtefactRetention,
	} {
		definition, ok := LookupSetting(name)
		require.True(t, ok, "%s is not in the registry", name)

		// A default that fails its own validation would leave a fresh
		// instance with a setting nobody can read.
		assert.NoError(t, definition.ValidateSetting(definition.Default))
	}
}
