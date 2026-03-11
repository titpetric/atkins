package model

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

// Iterator represents a single for-loop iteration specification,
// using format "item in items" or "(idx, item) in items".
type Iterator string

// Iterators is a slice of Iterator with custom YAML unmarshalling
// to support both single string and list of strings.
type Iterators []Iterator

// UnmarshalYAML implements custom unmarshalling for Iterators,
// supporting single string ("item in items") or list of strings.
func (its *Iterators) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string: "item in items"
		*its = Iterators{Iterator(node.Value)}
		return nil

	case yaml.SequenceNode:
		// List of strings
		var items []string
		if err := node.Decode(&items); err != nil {
			return fmt.Errorf("failed to decode for list: %w", err)
		}
		result := make(Iterators, len(items))
		for i, item := range items {
			result[i] = Iterator(item)
		}
		*its = result
		return nil

	default:
		return fmt.Errorf("invalid for format: expected string or list, got %v", node.Kind)
	}
}

// IsEmpty returns true if there are no iterators.
func (its Iterators) IsEmpty() bool {
	return len(its) == 0
}

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
