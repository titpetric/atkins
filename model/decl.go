package model

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

// Decl represents a common variables signature {vars, env, include}.
//
// It's a base type for pipelines, jobs/tasks and steps/cmds.
type Decl struct {
	Vars    map[string]any `yaml:"vars,omitempty"`
	Include *IncludeDecl   `yaml:"include,omitempty"`
	Env     *EnvDecl       `yaml:"env,omitempty"`
}

// EnvDecl represents an environment variable declaration that can contain
// both manually-set variables and includes from external files.
type EnvDecl Decl

// IncludeDecl represents file includes that can be either a single string or a list of strings.
type IncludeDecl struct {
	Files []string
}

// UnmarshalYAML implements custom unmarshalling for IncludeDecl to support string or []string.
func (e *IncludeDecl) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		// Single string
		e.Files = []string{node.Value}
		return nil
	}

	if node.Kind == yaml.SequenceNode {
		// List of strings
		var files []string
		if err := node.Decode(&files); err != nil {
			return err
		}
		e.Files = files
		return nil
	}

	return fmt.Errorf("invalid include format: expected string or list of strings, got %v", node.Kind)
}
