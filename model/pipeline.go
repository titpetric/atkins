package model

import (
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// Pipeline represents the root structure of an atkins.yml file.
type Pipeline struct {
	*Decl

	ID   string `yaml:"-"`
	Name string `yaml:"name,omitempty"`
	Dir  string `yaml:"dir,omitempty"`

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

// GetJobs will returned the defined jobs in the pipeline.
func (p *Pipeline) GetJobs() map[string]*Job {
	if len(p.Jobs) > 0 {
		return p.Jobs
	}
	return p.Tasks
}

// GetKeys will return the available targets in the pipeline. It uses the
// pipeline ID to optionally prefix job/tasks map keys. The default
// job is ordered first in the result.
func (p *Pipeline) GetKeys() []string {
	var hasDefault bool
	jobs := p.GetJobs()
	result := make([]string, 0, len(jobs))

	for key := range jobs {
		if key == "default" {
			hasDefault = true
			continue
		}

		result = append(result, key)
	}

	sort.Strings(result)

	if hasDefault {
		result = append([]string{"default"}, result...)
	}

	if p.ID != "" {
		for k, v := range result {
			result[k] = p.ID + ":" + v
		}
	}

	return result
}

// GetAliases will give key => value mapping for commands in a pipeline.
func (p *Pipeline) GetAliases() map[string]string {
	result := map[string]string{}
	jobs := p.GetJobs()

	for key, job := range jobs {
		// Alias skills default targets to skill ID.
		if p.ID != "" && key == "default" {
			result[p.ID] = "default"
		}

		// Add any other known aliases for the jobs.
		for _, alias := range job.GetAliases() {
			result[alias] = key
		}
	}

	// Prefix targets with skill ID.
	if p.ID != "" {
		for k, v := range result {
			result[k] = p.ID + ":" + v
		}
	}

	return result
}
