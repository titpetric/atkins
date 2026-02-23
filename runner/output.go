package runner

import (
	"encoding/json"
	"fmt"
	"sort"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ListOutputItem represents a single command in the list output.
type ListOutputItem struct {
	ID   string `json:"id" yaml:"id"`
	Desc string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Cmd  string `json:"cmd" yaml:"cmd"`
}

// ListOutputSection represents a pipeline section in the list output.
type ListOutputSection struct {
	Desc string           `json:"desc" yaml:"desc"`
	Cmds []ListOutputItem `json:"cmds" yaml:"cmds"`
}

// ListPipelinesJSON outputs pipelines in JSON format.
func ListPipelinesJSON(pipelines []*model.Pipeline) error {
	output := buildListOutput(pipelines)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// ListPipelinesYAML outputs pipelines in YAML format.
func ListPipelinesYAML(pipelines []*model.Pipeline) error {
	output := buildListOutput(pipelines)
	data, err := yaml.Marshal(output)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

// buildListOutput builds the structured list output from pipelines.
func buildListOutput(pipelines []*model.Pipeline) []ListOutputSection {
	if len(pipelines) == 0 {
		return nil
	}

	main, skills := separatePipelines(pipelines)
	var sections []ListOutputSection

	// Main pipeline section
	if main != nil && hasJobs(main) {
		sections = append(sections, buildPipelineSection(main, ""))
	}

	// Aliases section
	if aliases := buildAliasesSection(skills); len(aliases.Cmds) > 0 {
		sections = append(sections, aliases)
	}

	// Skill pipelines
	for _, skill := range skills {
		if hasJobs(skill) {
			sections = append(sections, buildPipelineSection(skill, skill.ID))
		}
	}

	return sections
}

// buildPipelineSection builds a section for a pipeline.
func buildPipelineSection(p *model.Pipeline, prefix string) ListOutputSection {
	jobs := getJobs(p)
	names := treeview.SortJobsByDepth(jobNames(jobs))

	// Move "default" to front
	for i, name := range names {
		if name == "default" {
			names = append([]string{name}, append(names[:i], names[i+1:]...)...)
			break
		}
	}

	var cmds []ListOutputItem
	for _, name := range names {
		job := jobs[name]

		id := name
		if prefix != "" {
			id = prefix + ":" + name
		}

		cmds = append(cmds, ListOutputItem{
			ID:   id,
			Desc: job.Desc,
			Cmd:  "atkins " + id,
		})
	}

	return ListOutputSection{
		Desc: p.Name,
		Cmds: cmds,
	}
}

// buildAliasesSection builds the aliases section.
func buildAliasesSection(skills []*model.Pipeline) ListOutputSection {
	var cmds []ListOutputItem

	for _, p := range skills {
		jobs := getJobs(p)

		// Skill ID alone is an alias to skill:default if default job exists
		if _, hasDefault := jobs["default"]; hasDefault {
			cmds = append(cmds, ListOutputItem{
				ID:  p.ID,
				Cmd: "atkins " + p.ID,
			})
		}

		// Collect explicit aliases from all jobs
		for jobName, job := range jobs {
			for _, alias := range job.Aliases {
				target := p.ID + ":" + jobName
				if jobName == "default" {
					target = p.ID
				}
				cmds = append(cmds, ListOutputItem{
					ID:   alias,
					Desc: fmt.Sprintf("invokes %s", target),
					Cmd:  "atkins " + alias,
				})
			}
		}
	}

	// Sort aliases
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].ID < cmds[j].ID
	})

	return ListOutputSection{
		Desc: "Aliases",
		Cmds: cmds,
	}
}
