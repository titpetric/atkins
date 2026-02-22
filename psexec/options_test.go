package psexec_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/psexec"
)

func TestDefaultOptions(t *testing.T) {
	opts := psexec.DefaultOptions()

	assert.NotNil(t, opts)
	assert.Equal(t, "bash", opts.DefaultShell)
}

func TestNewWithOptions(t *testing.T) {
	opts := &psexec.Options{
		DefaultDir: "/tmp",
		DefaultEnv: []string{"FOO=bar"},
	}

	exec := psexec.NewWithOptions(opts)
	assert.NotNil(t, exec)
}

func TestNewWithOptions_Nil(t *testing.T) {
	exec := psexec.NewWithOptions(nil)
	assert.NotNil(t, exec)
}
