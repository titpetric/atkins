package model

import (
	"fmt"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Iterator represents a single for-loop iteration specification,
// using format "item in items" or "(idx, item) in items". The alternative
// "items as item" syntax is normalized to "item in items" when decoded.
type Iterator string

// Iterators is a slice of Iterator with custom YAML unmarshalling
// to support both single string and list of strings.
type Iterators []Iterator

var iteratorAsPattern = regexp.MustCompile(`^(.+)\s+as\s+(\w+)\s*$`)

func normalizeIterator(value string) Iterator {
	if matches := iteratorAsPattern.FindStringSubmatch(strings.TrimSpace(value)); matches != nil {
		return Iterator(matches[2] + " in " + strings.TrimSpace(matches[1]))
	}
	return Iterator(value)
}

// UnmarshalYAML implements custom unmarshalling for Iterators,
// supporting single string ("item in items") or list of strings.
func (its *Iterators) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string: "item in items"
		*its = Iterators{normalizeIterator(node.Value)}
		return nil

	case yaml.SequenceNode:
		// List of strings
		var items []string
		if err := node.Decode(&items); err != nil {
			return fmt.Errorf("failed to decode for list: %w", err)
		}
		result := make(Iterators, len(items))
		for i, item := range items {
			result[i] = normalizeIterator(item)
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
