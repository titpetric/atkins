package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Field is one editable value in the configuration document.
type Field struct {
	// Path is the dotted document path, e.g. "server.lease_ttl".
	Path string

	// Env is the variable that overrides it at runtime, if any.
	Env string

	// Secret marks a value that should not be echoed back in full.
	Secret bool

	// value points at the live field so Set writes through.
	value reflect.Value
}

// Fields returns every editable field, in document order.
func (c *Config) Fields() []Field {
	var fields []Field
	collectFields(reflect.ValueOf(c).Elem(), "", &fields)
	return fields
}

// Field returns one field by path.
func (c *Config) Field(path string) (Field, bool) {
	for _, field := range c.Fields() {
		if field.Path == path {
			return field, true
		}
	}
	return Field{}, false
}

// secretFields are values worth masking in a listing: they are
// credentials, and a configuration menu is often on a shared screen.
var secretFields = map[string]bool{
	"server.signing_key": true,
	"server.agent_token": true,
	"agent.token":        true,
}

// collectFields walks a struct collecting its leaf values.
func collectFields(value reflect.Value, prefix string, fields *[]Field) {
	structType := value.Type()

	for i := range structType.NumField() {
		field := structType.Field(i)

		name := strings.ToLower(field.Name)
		if tag := field.Tag.Get("yaml"); tag != "" {
			name = strings.Split(tag, ",")[0]
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if field.Type.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Duration(0)) {
			collectFields(value.Field(i), path, fields)
			continue
		}

		// The document version is derived, not edited.
		if path == "version" {
			continue
		}

		*fields = append(*fields, Field{
			Path:   path,
			Env:    field.Tag.Get("env"),
			Secret: secretFields[path],
			value:  value.Field(i),
		})
	}
}

// String renders the current value for display.
func (f Field) String() string {
	switch typed := f.value.Interface().(type) {
	case time.Duration:
		return typed.String()
	case []string:
		return strings.Join(typed, ",")
	case bool:
		return strconv.FormatBool(typed)
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

// Display renders the value for a listing, masking secrets and marking
// empties so a blank line is never ambiguous.
func (f Field) Display() string {
	value := f.String()

	if value == "" {
		return "(not set)"
	}
	if f.Secret {
		return mask(value)
	}
	return value
}

// Kind names the value type, for the edit prompt.
func (f Field) Kind() string {
	switch f.value.Interface().(type) {
	case time.Duration:
		return "duration, e.g. 15m"
	case []string:
		return "comma separated list"
	case bool:
		return "true or false"
	}

	switch f.value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "whole number"
	default:
		return "text"
	}
}

// Set parses and assigns a value, reporting what didn't parse.
func (f Field) Set(raw string) error {
	raw = strings.TrimSpace(raw)

	switch f.value.Interface().(type) {
	case time.Duration:
		if raw == "" {
			f.value.Set(reflect.ValueOf(time.Duration(0)))
			return nil
		}
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("%s must be a duration such as 15m or 24h", f.Path)
		}
		f.value.Set(reflect.ValueOf(parsed))
		return nil

	case []string:
		f.value.Set(reflect.ValueOf(SplitList(raw)))
		return nil
	}

	switch f.value.Kind() {
	case reflect.String:
		f.value.SetString(raw)
		return nil

	case reflect.Bool:
		if raw == "" {
			f.value.SetBool(false)
			return nil
		}
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("%s must be true or false", f.Path)
		}
		f.value.SetBool(parsed)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if raw == "" {
			f.value.SetInt(0)
			return nil
		}
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be a whole number", f.Path)
		}
		f.value.SetInt(parsed)
		return nil
	}

	return fmt.Errorf("%s cannot be edited here", f.Path)
}

// mask shows enough of a secret to recognize it, and no more.
func mask(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
