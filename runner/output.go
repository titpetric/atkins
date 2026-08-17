package runner

import (
	"encoding/json"
	"fmt"
	"sort"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// OutputItem represents a single command in the list output.
type OutputItem struct {
	ID   string `json:"id" yaml:"id"`
	Desc string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Cmd  string `json:"cmd" yaml:"cmd"`

	// Interactive reports that the job binds stdin: either the job
	// declares `interactive: true` or one of its steps does.
	//
	// It is in the listing because the listing is what another program
	// reads to decide how to run a job. The CI server dispatches from
	// this and gives an interactive job a terminal that types back,
	// where everything else gets a transcript that only scrolls.
	Interactive bool `json:"interactive,omitempty" yaml:"interactive,omitempty"`

	// DependsOn are the jobs that run before this one. The pipeline has
	// carried them on model.Job all along; the listing did not, so a
	// caller reading the listing could not tell what a target drags in
	// with it.
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

// OutputSection represents a pipeline section in the list output.
type OutputSection struct {
	Desc string       `json:"desc" yaml:"desc"`
	Cmds []OutputItem `json:"cmds" yaml:"cmds"`
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
func buildListOutput(pipelines []*model.Pipeline) []OutputSection {
	if len(pipelines) == 0 {
		return nil
	}

	main, skills := separatePipelines(pipelines)
	var sections []OutputSection

	// Main pipeline section
	if main != nil && main.HasJobs() {
		sections = append(sections, buildPipelineSection(main, ""))
	}

	// Aliases section
	if aliases := buildAliasesSection(skills); len(aliases.Cmds) > 0 {
		sections = append(sections, aliases)
	}

	// Skill pipelines
	for _, skill := range skills {
		if skill.HasJobs() {
			sections = append(sections, buildPipelineSection(skill, skill.ID))
		}
	}

	return sections
}

// buildPipelineSection builds a section for a pipeline.
func buildPipelineSection(p *model.Pipeline, prefix string) OutputSection {
	jobs := p.GetJobs()
	names := treeview.SortJobsByDepth(p.JobNames())

	// Move "default" to front
	for i, name := range names {
		if name == "default" {
			names = append([]string{name}, append(names[:i], names[i+1:]...)...)
			break
		}
	}

	var cmds []OutputItem
	for _, name := range names {
		job := jobs[name]

		id := name
		if prefix != "" {
			id = prefix + ":" + name
		}

		cmds = append(cmds, OutputItem{
			ID:          id,
			Desc:        job.Desc,
			Cmd:         "atkins " + id,
			Interactive: bindsStdin(job),
			DependsOn:   job.DependsOn,
		})
	}

	return OutputSection{
		Desc: p.Name,
		Cmds: cmds,
	}
}

// bindsStdin reports whether running a job connects a terminal to it.
//
// A job says so for all of its steps, or one step says so for itself,
// and the executor treats the two the same way — see the isInteractive
// check in executor_command.go. The listing answers the question a
// caller actually has, which is "will this want a keyboard", not "which
// of the two places was it written in".
//
// It does not follow `task:` steps into the jobs they invoke. A job
// whose interactivity is somewhere down a chain of tasks is one nobody
// can read either, and guessing at it here would mean resolving names
// the listing does not otherwise resolve.
func bindsStdin(job *model.Job) bool {
	if job == nil {
		return false
	}
	if job.Interactive {
		return true
	}

	for _, step := range job.Children() {
		if step != nil && step.Interactive {
			return true
		}
	}

	return false
}

// buildAliasesSection builds the aliases section.
func buildAliasesSection(skills []*model.Pipeline) OutputSection {
	var cmds []OutputItem

	for _, p := range skills {
		jobs := p.GetJobs()

		// Skill ID alone is an alias to skill:default if default job exists
		if fallback, hasDefault := jobs["default"]; hasDefault {
			cmds = append(cmds, OutputItem{
				ID:          p.ID,
				Cmd:         "atkins " + p.ID,
				Interactive: bindsStdin(fallback),
			})
		}

		// Collect explicit aliases from all jobs
		for jobName, job := range jobs {
			for _, alias := range job.Aliases {
				target := p.ID + ":" + jobName
				if jobName == "default" {
					target = p.ID
				}
				cmds = append(cmds, OutputItem{
					ID:   alias,
					Desc: fmt.Sprintf("invokes %s", target),
					Cmd:  "atkins " + alias,
					// An alias runs the job it names, so it wants the same
					// terminal the job does.
					Interactive: bindsStdin(job),
				})
			}
		}
	}

	// Sort aliases
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].ID < cmds[j].ID
	})

	return OutputSection{
		Desc: "Aliases",
		Cmds: cmds,
	}
}
