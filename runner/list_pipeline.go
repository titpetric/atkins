package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ListPipelines displays pipelines grouped by section in a flat list format.
// The first pipeline is the default; subsequent ones are skill pipelines.
func ListPipelines(pipelines []*model.Pipeline) {
	if len(pipelines) == 0 {
		return
	}

	main := pipelines[0]
	printPipelineSection(main)

	// Skill pipelines sorted by name for consistent output.
	skills := make([]*model.Pipeline, len(pipelines)-1)
	copy(skills, pipelines[1:])
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	// Collect and print aliases from all skills
	printAliasesSection(skills)

	for _, p := range skills {
		printSkillSection(p)
	}
}

// aliasEntry represents a single alias mapping.
type aliasEntry struct {
	alias  string
	target string
}

// printAliasesSection prints all aliases from skill pipelines.
func printAliasesSection(skills []*model.Pipeline) {
	var aliases []aliasEntry

	for _, p := range skills {
		jobs := p.Jobs
		if len(jobs) == 0 {
			jobs = p.Tasks
		}

		// Add computed alias for skills with a "default" job
		if _, hasDefault := jobs["default"]; hasDefault {
			aliases = append(aliases, aliasEntry{alias: p.ID, target: p.ID + ":default"})
		}

		for name, job := range jobs {
			for _, alias := range job.Aliases {
				// Build full target name (e.g., "release:build")
				var target string
				if name == "default" {
					target = p.ID
				} else {
					target = p.ID + ":" + name
				}
				aliases = append(aliases, aliasEntry{alias: alias, target: target})
			}
		}
	}

	if len(aliases) == 0 {
		return
	}

	// Sort aliases alphabetically
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].alias < aliases[j].alias
	})

	// Find max alias length for alignment
	maxLen := 0
	for _, a := range aliases {
		if len(a.alias) > maxLen {
			maxLen = len(a.alias)
		}
	}

	fmt.Println()
	fmt.Printf("%s\n\n", colors.BrightWhite("Aliases"))

	for _, a := range aliases {
		padding := maxLen - len(a.alias) + 2
		fmt.Printf("* %s:%*s(invokes: %s)\n",
			colors.BrightGreen(a.alias),
			padding, "",
			colors.BrightOrange(a.target))
	}
}

// printPipelineSection prints a single pipeline section with all its jobs.
func printPipelineSection(p *model.Pipeline) {
	jobs := p.Jobs
	if len(jobs) == 0 {
		jobs = p.Tasks
	}

	// Main pipeline has no ID, skills have IDs
	isMain := p.ID == ""

	fmt.Printf("%s\n\n", colors.BrightWhite(p.Name))
	printJobList(jobs, p.ID, isMain)
}

// printSkillSection prints a skill pipeline.
// Job names are prefixed with the pipeline ID (e.g., "go:test" for skill "go").
func printSkillSection(p *model.Pipeline) {
	jobs := p.Jobs
	if len(jobs) == 0 {
		jobs = p.Tasks
	}

	if len(jobs) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("%s\n\n", colors.BrightWhite(p.Name))
	printJobList(jobs, p.ID, false) // Skills are never main
}

// printJobList prints a flat list of jobs with aligned descriptions.
// If prefix is non-empty, job names are displayed as "prefix:name" (or just "prefix" for default).
// isMain indicates if this is the main pipeline (green targets) or a skill (orange targets).
func printJobList(jobs map[string]*model.Job, prefix string, isMain bool) {
	names := treeview.SortJobsByDepth(jobNames(jobs))

	// Move "default" to the front if present
	for i, name := range names {
		if name == "default" {
			names = append([]string{name}, append(names[:i], names[i+1:]...)...)
			break
		}
	}

	// Build display names with optional prefix
	displayNames := make([]string, len(names))
	for i, name := range names {
		if prefix != "" {
			displayNames[i] = prefix + ":" + name
		} else {
			displayNames[i] = name
		}
	}

	// Find max name length for alignment.
	maxLen := 0
	for _, name := range displayNames {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	for i, name := range names {
		job := jobs[name]
		desc := job.Desc
		displayName := displayNames[i]
		padding := maxLen - len(displayName) + 2

		// Color: main pipeline targets are green, skill targets are orange
		var coloredName string
		if isMain {
			coloredName = colors.BrightGreen(displayName)
		} else {
			coloredName = colors.BrightOrange(displayName)
		}

		// Build depends_on suffix if present
		var depsStr string
		if deps := GetDependencies(job.DependsOn); len(deps) > 0 {
			depItems := make([]string, len(deps))
			for j, dep := range deps {
				depItems[j] = colors.BrightOrange(dep)
			}
			depsStr = fmt.Sprintf(" (depends_on: %s)", strings.Join(depItems, ", "))
		}

		// Build aliases suffix if present (only for main pipeline)
		var aliasStr string
		if isMain && len(job.Aliases) > 0 {
			aliasItems := make([]string, len(job.Aliases))
			for j, alias := range job.Aliases {
				aliasItems[j] = colors.BrightGreen(alias)
			}
			aliasStr = fmt.Sprintf(" (aliases: %s)", strings.Join(aliasItems, ", "))
		}

		if desc != "" {
			fmt.Printf("* %s:%*s%s%s%s\n", coloredName, padding, "", desc, depsStr, aliasStr)
		} else if depsStr != "" || aliasStr != "" {
			fmt.Printf("* %s:%*s%s%s\n", coloredName, padding, "", depsStr, aliasStr)
		} else {
			fmt.Printf("* %s:\n", coloredName)
		}
	}
}

// jobNames returns the keys of a job map.
func jobNames(jobs map[string]*model.Job) []string {
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}
	return names
}
