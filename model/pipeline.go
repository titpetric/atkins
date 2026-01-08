package model

import (
	yaml "gopkg.in/yaml.v3"
)

// Pipeline represents the root structure of an atkins.yml file.
type Pipeline struct {
	*Decl

	Name  string          `yaml:"name,omitempty"`
	Jobs  map[string]*Job `yaml:"jobs,omitempty"`
	Tasks map[string]*Job `yaml:"tasks,omitempty"`
}

// UnmarshalYAML implements custom unmarshalling for Pipeline to handle Decl.
func (p *Pipeline) UnmarshalYAML(node *yaml.Node) error {
	type rawPipeline Pipeline
	if err := node.Decode((*rawPipeline)(p)); err != nil {
		return err
	}

	// Ensure Decl is initialized and vars/include are properly decoded
	if err := ensureDeclInitialized(&p.Decl, node); err != nil {
		return err
	}

	return nil
}
