package config_test

import (
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

func TestDefaultValidates(t *testing.T) {
	t.Parallel()

	cfg, err := config.Default()
	assert.NoError(t, err)
	assert.NoError(t, cfg.Validate())
}

func TestValidateFillsEmptyValues(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	assert.NoError(t, cfg.Validate())

	assert.Equal(t, config.Version, cfg.Version)
}

func TestValidateRejectsAVersionFromTheFuture(t *testing.T) {
	t.Parallel()

	cfg, err := config.Default()
	assert.NoError(t, err)

	// A newer build may have written fields this one would ignore in
	// silence, which is worse than refusing to start.
	cfg.Version = config.Version + 1
	assert.Error(t, cfg.Validate())
}
