package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/server/model"
)

func TestMatchRepository(t *testing.T) {
	tests := []struct {
		pattern  string
		slug     string
		expected bool
	}{
		{"github.com/titpetric/atkins", "github.com/titpetric/atkins", true},
		{"github.com/titpetric/atkins", "github.com/titpetric/pdo", false},

		{"github.com/titpetric/*", "github.com/titpetric/atkins", true},
		{"github.com/titpetric/*", "github.com/someone/atkins", false},
		// `*` stays inside one segment.
		{"github.com/*", "github.com/titpetric/atkins", false},

		{"github.com/**", "github.com/titpetric/atkins", true},
		{"github.com/**", "github.com/titpetric", true},
		{"github.com/**", "gitlab.com/titpetric/atkins", false},
		{"**", "anything/at/all", true},

		{"github.com/titpetric/atkins-*", "github.com/titpetric/atkins-ci", true},
		{"github.com/titpetric/atkins-*", "github.com/titpetric/atkins", false},

		// Slugs are lowercased, and so are patterns.
		{"GitHub.com/TitPetric/*", "github.com/titpetric/atkins", true},

		{"", "github.com/titpetric/atkins", false},
		{"github.com/**", "", false},

		// A pattern must consume the whole slug.
		{"github.com/titpetric", "github.com/titpetric/atkins", false},
	}

	for _, test := range tests {
		t.Run(test.pattern+" vs "+test.slug, func(t *testing.T) {
			assert.Equal(t, test.expected, model.MatchRepository(test.pattern, test.slug))
		})
	}
}

func TestValidRepositoryPolicy(t *testing.T) {
	assert.True(t, model.ValidRepositoryPolicy(model.PolicyOpen))
	assert.True(t, model.ValidRepositoryPolicy(model.PolicyAllowlist))
	assert.False(t, model.ValidRepositoryPolicy("permissive"))
}

func TestSettingDefinitions(t *testing.T) {
	definitions := model.SettingDefinitions()
	assert.NotEmpty(t, definitions)

	policy, ok := model.LookupSetting(model.SettingRepositoryPolicy)
	assert.True(t, ok)
	assert.Equal(t, model.PolicyOpen, policy.Default)

	_, ok = model.LookupSetting("nonsense")
	assert.False(t, ok)
}

func TestValidateSetting(t *testing.T) {
	policy, _ := model.LookupSetting(model.SettingRepositoryPolicy)
	assert.NoError(t, policy.ValidateSetting(model.PolicyAllowlist))
	assert.Error(t, policy.ValidateSetting("permissive"))

	depth, _ := model.LookupSetting(model.SettingJobMaxDepth)
	assert.NoError(t, depth.ValidateSetting("5"))
	assert.Error(t, depth.ValidateSetting("five"))

	lease, _ := model.LookupSetting(model.SettingJobLeaseTTL)
	assert.NoError(t, lease.ValidateSetting("15m"))
	assert.Error(t, lease.ValidateSetting("fifteen"))

	open, _ := model.LookupSetting(model.SettingRegistrationOpen)
	assert.NoError(t, open.ValidateSetting("true"))
	assert.Error(t, open.ValidateSetting("yes please"))
}
