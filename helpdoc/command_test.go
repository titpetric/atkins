package helpdoc_test

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/titpetric/atkins/helpdoc"
	"github.com/titpetric/atkins/internal/assert"
)

// TestCommand checks a command carries the flag set it was given.
func TestCommand(t *testing.T) {
	flags := pflag.NewFlagSet("server", pflag.ContinueOnError)
	var addr string
	flags.StringVar(&addr, "addr", ":3000", "Listen address")

	command := helpdoc.Command{
		Name:  "server",
		Title: "Run the CI/CD server",
		Flags: flags,
	}

	assert.Equal(t, "server", command.Name)
	assert.False(t, command.Default)
	assert.True(t, command.Flags.HasFlags())
	assert.NotNil(t, command.Flags.Lookup("addr"))
}
