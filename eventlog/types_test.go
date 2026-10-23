package eventlog_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
)

// An event keeps its type through the YAML document of --log, and a
// step event leaves the field out.
func TestEventType(t *testing.T) {
	t.Parallel()

	for _, eventType := range []eventlog.EventType{
		eventlog.EventTypeSubstitution,
		eventlog.EventTypeInterpolation,
	} {
		out, err := yaml.Marshal(&eventlog.Event{ID: "e", Type: eventType})
		assert.NoError(t, err)

		var back eventlog.Event
		assert.NoError(t, yaml.Unmarshal(out, &back))
		assert.Equal(t, eventType, back.Type)
	}

	out, err := yaml.Marshal(&eventlog.Event{ID: "e"})
	assert.NoError(t, err)
	assert.NotContains(t, string(out), "type:")
}
