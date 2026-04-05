package runner

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ListPipelines returns pipelines formatted as a string in a flat list format:
// Main Pipeline, then Aliases, then Skills.
func ListPipelines(pipelines []*model.Pipeline) string {
	if len(pipelines) == 0 {
		return ""
	}

	main, skills := separatePipelines(pipelines)

	var sections []string
	if s := formatPipelineSection(main); s != "" {
		sections = append(sections, s)
	}
	if s := formatAliasesSection(skills); s != "" {
		sections = append(sections, s)
	}
	for _, skill := range skills {
		if s := formatPipelineSection(skill); s != "" {
			sections = append(sections, s)
		}
	}

	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n") + "\n"
}

// separatePipelines divides pipelines into main (ID="") and skills (ID!="").
func separatePipelines(pipelines []*model.Pipeline) (*model.Pipeline, []*model.Pipeline) {
	var main *model.Pipeline
	var skills []*model.Pipeline

	for _, p := range pipelines {
		if p.ID == "" {
			main = p
		} else {
			skills = append(skills, p)
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return main, skills
}

// formatPipelineSection formats a pipeline header and its job list.
func formatPipelineSection(p *model.Pipeline) string {
	if p == nil {
		return ""
	}
	if !p.HasJobs() {
		return ""
	}

	return fmt.Sprintf("%s\n\n%s", colors.BrightWhite(p.Name), strings.Join(formatJobLines(p.GetJobs(), p.ID), "\n"))
}

// formatAliasesSection collects and formats all aliases from skill pipelines.
func formatAliasesSection(skills []*model.Pipeline) string {
	type aliasEntry struct {
		alias  string
		target string
	}

	var aliases []aliasEntry
	for _, p := range skills {
		jobs := p.GetJobs()
		if _, hasDefault := jobs["default"]; hasDefault {
			aliases = append(aliases, aliasEntry{p.ID, p.ID + ":default"})
		}
		for jobName, job := range jobs {
			for _, alias := range job.Aliases {
				target := p.ID + ":" + jobName
				if jobName == "default" {
					target = p.ID
				}
				aliases = append(aliases, aliasEntry{alias, target})
			}
		}
	}
	if len(aliases) == 0 {
		return ""
	}

	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].alias < aliases[j].alias
	})

	maxLen := 0
	for _, a := range aliases {
		if len(a.alias) > maxLen {
			maxLen = len(a.alias)
		}
	}

	lines := make([]string, len(aliases))
	for i, a := range aliases {
		padding := maxLen - len(a.alias) + 2
		lines[i] = fmt.Sprintf("* %s:%*s(invokes: %s)",
			colors.BrightGreen(a.alias), padding, "",
			colors.BrightOrange(a.target))
	}

	return fmt.Sprintf("%s\n\n%s", colors.BrightWhite("Aliases"), strings.Join(lines, "\n"))
}

// formatJobLines produces a formatted line per job with description, deps, and aliases.
func formatJobLines(jobs map[string]*model.Job, prefix string) []string {
	names := treeview.SortJobsByDepth(slices.Collect(maps.Keys(jobs)))
	for i, name := range names {
		if name == "default" {
			names = append([]string{name}, append(names[:i], names[i+1:]...)...)
			break
		}
	}

	isMain := prefix == ""
	displayNames := make([]string, len(names))
	for i, name := range names {
		if prefix != "" {
			displayNames[i] = prefix + ":" + name
		} else {
			displayNames[i] = name
		}
	}

	maxLen := 0
	for _, dn := range displayNames {
		if len(dn) > maxLen {
			maxLen = len(dn)
		}
	}

	lines := make([]string, len(names))
	for i, jobName := range names {
		job := jobs[jobName]
		dn := displayNames[i]
		padding := maxLen - len(dn) + 2

		coloredName := colors.BrightOrange(dn)
		if isMain {
			coloredName = colors.BrightGreen(dn)
		}

		depsStr := formatDependsOn(job)
		aliasStr := ""
		if isMain && len(job.Aliases) > 0 {
			items := make([]string, len(job.Aliases))
			for j, a := range job.Aliases {
				items[j] = colors.BrightGreen(a)
			}
			aliasStr = fmt.Sprintf(" (aliases: %s)", strings.Join(items, ", "))
		}

		switch {
		case job.Desc != "":
			lines[i] = fmt.Sprintf("* %s:%*s%s%s%s", coloredName, padding, "", job.Desc, depsStr, aliasStr)
		case depsStr != "" || aliasStr != "":
			lines[i] = fmt.Sprintf("* %s%*s%s%s", coloredName, padding+1, "", depsStr, aliasStr)
		default:
			lines[i] = fmt.Sprintf("* %s", coloredName)
		}
	}
	return lines
}

func formatDependsOn(job *model.Job) string {
	deps := GetDependencies(job.DependsOn)
	if len(deps) == 0 {
		return ""
	}
	items := make([]string, len(deps))
	for i, dep := range deps {
		items[i] = colors.BrightOrange(dep)
	}
	return fmt.Sprintf(" (depends_on: %s)", strings.Join(items, ", "))
}
