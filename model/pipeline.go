package model

import (
	yaml "gopkg.in/yaml.v3"
)

// Pipeline represents the root structure of an atkins.yml file.
type Pipeline struct {
	*Decl

	ID   string `yaml:"-"`
	Name string `yaml:"name,omitempty"`

	Jobs  map[string]*Job `yaml:"jobs,omitempty"`
	Tasks map[string]*Job `yaml:"tasks,omitempty"`

	When *PipelineWhen `yaml:"when,omitempty"`
}

// UnmarshalYAML implements custom unmarshalling for Pipeline to handle Decl.
func (p *Pipeline) UnmarshalYAML(node *yaml.Node) error {
	type rawPipeline Pipeline
	if err := node.Decode((*rawPipeline)(p)); err != nil {
		return err
	}

	// Ensure Decl is initialized and vars/include are properly decoded
	if err := ensureDeclInitialized(node, &p.Decl); err != nil {
		return err
	}

	return nil
}
