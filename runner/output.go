package runner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// Section kinds in the list output.
const (
	KindPipeline = "pipeline"
	KindSkill    = "skill"
	KindAliases  = "aliases"
)

// OutputWhen reports a when: block in the list output and the file that
// satisfied it.
type OutputWhen struct {
	Files    []string `json:"files" yaml:"files"`
	Here     bool     `json:"here,omitempty" yaml:"here,omitempty"`
	Contains string   `json:"contains,omitempty" yaml:"contains,omitempty"`
	Matched  string   `json:"matched,omitempty" yaml:"matched,omitempty"`
}

// OutputItem represents a single command in the list output.
type OutputItem struct {
	ID string `json:"id" yaml:"id"`

	// Name is the job name inside its pipeline, without the skill
	// prefix, or the alias itself.
	Name string `json:"name" yaml:"name"`

	Desc string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Cmd  string `json:"cmd" yaml:"cmd"`

	// Interactive reports a job that reads from the terminal.
	Interactive bool `json:"interactive" yaml:"interactive"`

	// Args reports a job that takes arguments after a bare --.
	Args bool `json:"args" yaml:"args"`

	// Exec is the argv of the first exec: step of a job that takes
	// arguments, as the file writes it: the command the arguments
	// follow, for a caller to show before it runs the job.
	Exec []string `json:"exec,omitempty" yaml:"exec,omitempty"`

	// Active is false for a job of an inactive skill, which only a
	// listing with --all includes. A job whose own when: block does not
	// match never lists.
	Active bool `json:"active" yaml:"active"`

	When *OutputWhen `json:"when,omitempty" yaml:"when,omitempty"`
}

// OutputSection represents a pipeline section in the list output.
type OutputSection struct {
	// ID is the skill namespace, empty for the main pipeline and the
	// aliases.
	ID string `json:"id" yaml:"id"`

	// Kind is one of pipeline, skill or aliases.
	Kind string `json:"kind" yaml:"kind"`

	// Path is the file the section loads from, empty for the aliases
	// and for a pipeline read from stdin.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Dir is the folder the section's jobs run in, empty for the
	// aliases and for an inactive skill.
	Dir string `json:"dir,omitempty" yaml:"dir,omitempty"`

	// Active is false for a skill whose when: block does not match.
	// Only a listing with --all includes one.
	Active bool `json:"active" yaml:"active"`

	When *OutputWhen `json:"when,omitempty" yaml:"when,omitempty"`

	Desc string       `json:"desc" yaml:"desc"`
	Cmds []OutputItem `json:"cmds" yaml:"cmds"`
}

// OutputOptions adjusts the list output.
type OutputOptions struct {
	// All includes the skills whose when: block does not match, with
	// active: false.
	All bool

	// Dir is the folder the main pipeline runs in when it sets no
	// dir: of its own. Empty means the folder of its file.
	Dir string
}

// OutputError is the list output for a pipeline that fails the linter.
type OutputError struct {
	Error string            `json:"error" yaml:"error"`
	Lint  []OutputLintError `json:"lint" yaml:"lint"`
}

// OutputLintError is one linter finding in OutputError.
type OutputLintError struct {
	Pipeline string `json:"pipeline" yaml:"pipeline"`
	Path     string `json:"path,omitempty" yaml:"path,omitempty"`
	Job      string `json:"job" yaml:"job"`
	Issue    string `json:"issue" yaml:"issue"`
	Detail   string `json:"detail" yaml:"detail"`
}

// NewOutputError builds the list output for lint findings in a pipeline.
func NewOutputError(pipeline *model.Pipeline, lintErrors []LintError) OutputError {
	result := OutputError{
		Error: fmt.Sprintf("pipeline '%s' has errors", pipeline.Name),
		Lint:  make([]OutputLintError, 0, len(lintErrors)),
	}
	lintErrors = slices.Clone(lintErrors)
	sort.SliceStable(lintErrors, func(i, j int) bool {
		return lintErrors[i].Job < lintErrors[j].Job
	})
	for _, lintErr := range lintErrors {
		result.Lint = append(result.Lint, OutputLintError{
			Pipeline: pipeline.Name,
			Path:     pipeline.Path,
			Job:      lintErr.Job,
			Issue:    lintErr.Issue,
			Detail:   lintErr.Detail,
		})
	}
	return result
}

// PrintJSON writes v as indented JSON to stdout.
func PrintJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintYAML writes v as YAML to stdout.
func PrintYAML(v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

// ListPipelinesJSON outputs pipelines in JSON format.
func ListPipelinesJSON(pipelines []*model.Pipeline, opts ...OutputOptions) error {
	return PrintJSON(buildListOutput(pipelines, opts...))
}

// ListPipelinesYAML outputs pipelines in YAML format.
func ListPipelinesYAML(pipelines []*model.Pipeline, opts ...OutputOptions) error {
	return PrintYAML(buildListOutput(pipelines, opts...))
}

// ActivePipelines returns the pipelines a run sees. It drops the
// inactive skills, and returns a copy without the inactive jobs of a
// pipeline that has some.
func ActivePipelines(pipelines []*model.Pipeline) []*model.Pipeline {
	result := make([]*model.Pipeline, 0, len(pipelines))
	for _, p := range pipelines {
		if p.Inactive {
			continue
		}
		result = append(result, withoutInactiveJobs(p))
	}
	return result
}

// withoutInactiveJobs returns p, or a shallow copy of p whose job map
// leaves out the inactive jobs.
func withoutInactiveJobs(p *model.Pipeline) *model.Pipeline {
	jobs := p.GetJobs()
	inactive := false
	for _, job := range jobs {
		if job != nil && job.Inactive {
			inactive = true
			break
		}
	}
	if !inactive {
		return p
	}

	active := make(map[string]*model.Job, len(jobs))
	for name, job := range jobs {
		if job == nil || !job.Inactive {
			active[name] = job
		}
	}

	clone := *p
	if len(p.Jobs) > 0 {
		clone.Jobs = active
	} else {
		clone.Tasks = active
	}
	return &clone
}

// buildListOutput builds the structured list output from pipelines.
func buildListOutput(pipelines []*model.Pipeline, opts ...OutputOptions) []OutputSection {
	if len(pipelines) == 0 {
		return nil
	}

	var opt OutputOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	// A job whose when: block does not match never lists; --all brings
	// back the inactive skills only.
	listed := make([]*model.Pipeline, 0, len(pipelines))
	for _, p := range pipelines {
		if p.Inactive && !opt.All {
			continue
		}
		listed = append(listed, withoutInactiveJobs(p))
	}

	main, skills := separatePipelines(listed)
	var sections []OutputSection

	// Main pipeline section
	if main != nil && main.HasJobs() {
		section := buildPipelineSection(main, "")
		section.Kind = KindPipeline
		section.Path = main.Path
		section.Dir = pipelineDir(main, opt.Dir)
		section.Active = true
		sections = append(sections, section)
	}

	// Aliases section
	if aliases := buildAliasesSection(ActivePipelines(skills)); len(aliases.Cmds) > 0 {
		sections = append(sections, aliases)
	}

	// Skill pipelines
	for _, skill := range skills {
		if skill.HasJobs() {
			section := buildPipelineSection(skill, skill.ID)
			section.ID = skill.ID
			section.Kind = KindSkill
			section.Path = skill.Path
			section.Active = !skill.Inactive
			section.When = outputWhen(skill.When, skill.Matched)
			if section.Active {
				section.Dir = pipelineDir(skill, "")
			}
			sections = append(sections, section)
		}
	}

	return sections
}

// pipelineDir returns the folder the pipeline's jobs run in: its own
// dir:, resolved against the folder of its file when relative, or
// fallback, or the folder of its file.
func pipelineDir(p *model.Pipeline, fallback string) string {
	base := fallback
	if base == "" && p.Path != "" {
		base = filepath.Dir(p.Path)
	}
	switch {
	case p.Dir == "":
		return base
	case filepath.IsAbs(p.Dir) || base == "":
		return p.Dir
	default:
		return filepath.Join(base, p.Dir)
	}
}

// outputWhen converts a when: block for the list output.
func outputWhen(when *model.PipelineWhen, matched string) *OutputWhen {
	if when.IsEmpty() {
		return nil
	}
	return &OutputWhen{
		Files:    when.Files,
		Here:     when.Here,
		Contains: when.Contains,
		Matched:  matched,
	}
}

// argsExec returns the argv of the first exec: step of a job that takes
// arguments, nil for any other job.
func argsExec(job *model.Job) []string {
	if job == nil || !job.Args {
		return nil
	}
	for _, step := range job.Children() {
		if step != nil && len(step.Exec) > 0 {
			return step.Exec
		}
	}
	return nil
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
			Name:        name,
			Interactive: job.Interactive,
			Args:        job.Args,
			Exec:        argsExec(job),
			Active:      !p.Inactive,
			When:        outputWhen(job.When, job.Matched),
		})
	}

	return OutputSection{
		Desc: p.Name,
		Cmds: cmds,
	}
}

// buildAliasesSection builds the aliases section.
func buildAliasesSection(skills []*model.Pipeline) OutputSection {
	var cmds []OutputItem

	for _, p := range skills {
		jobs := p.GetJobs()

		// Skill ID alone is an alias to skill:default if default job exists
		if job, hasDefault := jobs["default"]; hasDefault {
			item := OutputItem{
				ID:     p.ID,
				Cmd:    "atkins " + p.ID,
				Name:   p.ID,
				Active: true,
			}
			if job != nil {
				item.Interactive, item.Args = job.Interactive, job.Args
			}
			cmds = append(cmds, item)
		}

		// Collect explicit aliases from all jobs
		for jobName, job := range jobs {
			for _, alias := range job.Aliases {
				target := p.ID + ":" + jobName
				if jobName == "default" {
					target = p.ID
				}
				cmds = append(cmds, OutputItem{
					ID:          alias,
					Desc:        fmt.Sprintf("invokes %s", target),
					Cmd:         "atkins " + alias,
					Name:        alias,
					Interactive: job.Interactive,
					Args:        job.Args,
					Active:      true,
				})
			}
		}
	}

	// Sort aliases
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].ID < cmds[j].ID
	})

	return OutputSection{
		Desc:   "Aliases",
		Cmds:   cmds,
		Kind:   KindAliases,
		Active: true,
	}
}
