package model

import (
	"maps"
	"slices"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// Pipeline represents the root structure of an atkins.yml file.
type Pipeline struct {
	*Decl

	ID   string `yaml:"-"`
	Name string `yaml:"name,omitempty"`
	Dir  string `yaml:"dir,omitempty"`

	// Help is one or two sentences on what the pipeline is for, and the
	// files it reads and writes. `atkins --help` prints it under the
	// skill the pipeline came from.
	Help string `yaml:"help,omitempty"`

	Jobs  map[string]*Job `yaml:"jobs,omitempty"`
	Tasks map[string]*Job `yaml:"tasks,omitempty"`

	When *PipelineWhen `yaml:"when,omitempty"`

	// Path is the file the pipeline loads from, empty for stdin.
	Path string `yaml:"-"`

	// Matched is the file that satisfied the when: block of a skill,
	// empty when the skill has no when: block or it does not match.
	Matched string `yaml:"-"`

	// Inactive marks a skill whose when: block does not match. Only the
	// listing of -l --all carries one; it contributes no jobs to a run.
	Inactive bool `yaml:"-"`
}

// UnmarshalYAML implements custom unmarshalling for Pipeline to handle Decl.
func (p *Pipeline) UnmarshalYAML(node *yaml.Node) error {
	type rawPipeline Pipeline
	if err := node.Decode((*rawPipeline)(p)); err != nil {
		return err
	}

	// Ensure Decl exists and vars/include decode properly
	if err := ensureDeclInitialized(node, &p.Decl); err != nil {
		return err
	}

	return nil
}

// GetJobs returns Jobs, or Tasks when Jobs is empty.
func (p *Pipeline) GetJobs() map[string]*Job {
	if len(p.Jobs) > 0 {
		return p.Jobs
	}
	return p.Tasks
}

// GetKeys returns the sorted job names, "default" first, prefixed with "ID:" for a pipeline with an ID.
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

// HasJobs returns true if the pipeline has any jobs or tasks defined.
func (p *Pipeline) HasJobs() bool {
	return len(p.GetJobs()) > 0
}

// JobNames returns the names of all jobs (or tasks) in the pipeline.
func (p *Pipeline) JobNames() []string {
	return slices.Collect(maps.Keys(p.GetJobs()))
}

// GetAliases returns alias => job name, explicit job aliases ahead of the generated ones.
func (p *Pipeline) GetAliases() map[string]string {
	result := map[string]string{}
	explicit := make(map[string]bool)
	jobs := p.GetJobs()

	// First pass: collect explicit aliases (these take priority).
	for key, job := range jobs {
		for _, alias := range job.GetAliases() {
			result[alias] = key
			explicit[alias] = true
		}
	}

	// Second pass: add auto-aliases only if not already set by explicit alias.
	for key := range jobs {
		// Alias skills default targets to skill ID.
		if p.ID != "" && key == "default" {
			if !explicit[p.ID] {
				result[p.ID] = "default"
			}
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
