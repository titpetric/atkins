package config_test

import (
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

func TestEnvironmentNames(t *testing.T) {
	t.Parallel()

	names := config.EnvironmentNames()

	assert.Equal(t, "ATKINS_DB", names["storage.database"])
	assert.Equal(t, "ATKINS_RECORD", names["storage.record"])
}

func TestSplitList(t *testing.T) {
	t.Parallel()

	assert.Nil(t, config.SplitList(""))
	assert.Nil(t, config.SplitList(" , , "))
	assert.Equal(t, []string{"a", "b"}, config.SplitList(" a , b "))
}
