package helpdoc_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/helpdoc"
	"github.com/titpetric/atkins/model"
)

// TestSchema checks the embedded reference is present and trimmed.
func TestSchema(t *testing.T) {
	schema := helpdoc.Schema()

	require.NotEmpty(t, schema)
	assert.Equal(t, schema, strings.TrimSpace(schema))
	assert.Contains(t, schema, "### One idea, several spellings")
	assert.Contains(t, schema, "### Top level")
	assert.Contains(t, schema, "### A job")
	assert.Contains(t, schema, "### A step")
}

// TestSchemaDocumentsModelKeys reads the yaml tags off the model types
// and fails when the hand-written reference documents a key none of them
// declares. It is the check that keeps a hand-written document true to
// the schema `splint -schema -i model` prints.
func TestSchemaDocumentsModelKeys(t *testing.T) {
	sections := map[string]map[string]bool{
		"### Top level": union(
			yamlTags(reflect.TypeOf(model.Pipeline{})),
			yamlTags(reflect.TypeOf(model.Decl{})),
		),
		"### A job": union(
			yamlTags(reflect.TypeOf(model.Job{})),
			yamlTags(reflect.TypeOf(model.Decl{})),
		),
		"### A step": union(
			yamlTags(reflect.TypeOf(model.Step{})),
			yamlTags(reflect.TypeOf(model.Decl{})),
			yamlTags(reflect.TypeOf(model.DeferredStep{})),
		),
	}

	for heading, tags := range sections {
		keys := documentedKeys(t, helpdoc.Schema(), heading)
		require.NotEmpty(t, keys, "no keys documented under %s", heading)

		for _, key := range keys {
			assert.True(t, tags[key], "%s documents %q, which no model yaml tag declares", heading, key)
		}
	}
}

// documentedKeys returns the keys named in the first column of the table
// under a heading, up to the next heading.
func documentedKeys(t *testing.T, schema, heading string) []string {
	t.Helper()

	_, rest, found := strings.Cut(schema, heading)
	require.True(t, found, "heading %q not found", heading)

	if end := strings.Index(rest, "\n### "); end >= 0 {
		rest = rest[:end]
	}

	var keys []string
	for _, line := range strings.Split(rest, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}

		column, _, _ := strings.Cut(strings.TrimPrefix(line, "| "), " | ")
		for _, cell := range strings.Split(column, ",") {
			key := strings.Trim(strings.TrimSpace(cell), "`")
			if key != "" {
				keys = append(keys, key)
			}
		}
	}

	return keys
}

// yamlTags returns the yaml key of every field of a struct, descending
// into embedded types, and skipping the fields tagged out of the
// document with a dash.
func yamlTags(typ reflect.Type) map[string]bool {
	tags := map[string]bool{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		embedded := field.Type
		if embedded.Kind() == reflect.Ptr {
			embedded = embedded.Elem()
		}
		if field.Anonymous && embedded.Kind() == reflect.Struct {
			for key := range yamlTags(embedded) {
				tags[key] = true
			}
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			continue
		}
		tags[name] = true
	}

	return tags
}

// union merges tag sets.
func union(sets ...map[string]bool) map[string]bool {
	result := map[string]bool{}
	for _, set := range sets {
		for key := range set {
			result[key] = true
		}
	}
	return result
}
