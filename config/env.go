package config

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// EnvironmentNames returns the override variable of each field, keyed by field path.
func EnvironmentNames() map[string]string {
	names := map[string]string{}
	collectEnvironment(reflect.TypeOf(Config{}), "", names)
	return names
}

// applyEnvironment walks a struct, overlaying tagged fields.
func applyEnvironment(value reflect.Value, values map[string]string) {
	structType := value.Type()

	for i := range structType.NumField() {
		field := value.Field(i)
		if field.Kind() == reflect.Struct && field.Type() != reflect.TypeOf(time.Duration(0)) {
			applyEnvironment(field, values)
			continue
		}

		name := structType.Field(i).Tag.Get("env")
		if name == "" {
			continue
		}

		raw, ok := values[name]
		if !ok {
			continue
		}

		assign(field, raw)
	}
}

// collectEnvironment walks a struct type collecting env tags.
func collectEnvironment(structType reflect.Type, prefix string, names map[string]string) {
	for i := range structType.NumField() {
		field := structType.Field(i)

		path := strings.ToLower(field.Name)
		if tag := field.Tag.Get("yaml"); tag != "" {
			path = strings.Split(tag, ",")[0]
		}
		if prefix != "" {
			path = prefix + "." + path
		}

		if field.Type.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Duration(0)) {
			collectEnvironment(field.Type, path, names)
			continue
		}

		if name := field.Tag.Get("env"); name != "" {
			names[path] = name
		}
	}
}

// assign parses raw into field, leaving it alone if it doesn't parse.
func assign(field reflect.Value, raw string) {
	switch field.Interface().(type) {
	case time.Duration:
		if parsed, err := time.ParseDuration(raw); err == nil {
			field.Set(reflect.ValueOf(parsed))
		}
		return
	case []string:
		field.Set(reflect.ValueOf(SplitList(raw)))
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		if parsed, err := strconv.ParseBool(raw); err == nil {
			field.SetBool(parsed)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			field.SetInt(parsed)
		}
	}
}

// SplitList parses a comma separated value into a list, dropping empty
// entries so `a,,b` and ` a , b ` both return two items.
func SplitList(value string) []string {
	parts := strings.Split(value, ",")

	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			items = append(items, part)
		}
	}

	if len(items) == 0 {
		return nil
	}
	return items
}
