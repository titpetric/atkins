package model

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Condition represents a single if-condition expression.
type Condition string

// Conditionals is a slice of Condition with custom YAML unmarshalling
// to support both single string and list of strings.
type Conditionals []Condition

// UnmarshalYAML implements custom unmarshalling for Conditionals,
// supporting single string ("enabled == true") or list of strings.
func (cs *Conditionals) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string condition
		*cs = Conditionals{Condition(node.Value)}
		return nil

	case yaml.SequenceNode:
		// List of strings
		var items []string
		if err := node.Decode(&items); err != nil {
			return fmt.Errorf("failed to decode if list: %w", err)
		}
		result := make(Conditionals, len(items))
		for i, item := range items {
			result[i] = Condition(item)
		}
		*cs = result
		return nil

	default:
		return fmt.Errorf("invalid if format: expected string or list, got %v", node.Kind)
	}
}

// IsEmpty returns true if there are no conditions.
func (cs Conditionals) IsEmpty() bool {
	return len(cs) == 0
}

// String returns a display representation of the conditions.
// Single condition returns the expression as-is.
// Multiple conditions are parenthesized and joined with " && ".
func (cs Conditionals) String() string {
	if len(cs) == 0 {
		return ""
	}
	if len(cs) == 1 {
		return strings.TrimSpace(string(cs[0]))
	}
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = "(" + strings.TrimSpace(string(c)) + ")"
	}
	return strings.Join(parts, " && ")
}
