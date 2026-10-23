package runner_test

import (
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/runner"
)

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	var opts *runner.Options = runner.DefaultOptions()
	assert.Equal(t, 15*time.Minute, opts.DefaultTimeout)
}
