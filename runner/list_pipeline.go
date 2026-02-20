package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ListPipelines displays pipelines grouped by section in a flat list format:
// Main Pipeline, then Aliases, then Skills.
func ListPipelines(pipelines []*model.Pipeline) {
	if len(pipelines) == 0 {
		return
	}

	main, skills := separatePipelines(pipelines)

	mainPrinted := printMainPipeline(main)
	printAliasesSection(skills, mainPrinted)
	printSkillPipelines(skills)
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

	// Sort skills by name for consistent output
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return main, skills
}

// printMainPipeline prints the main pipeline if it has jobs. Returns true if printed.
func printMainPipeline(main *model.Pipeline) bool {
	if main == nil {
		return false
	}

	if !hasJobs(main) {
		return false
	}

	printPipelineSection(main)
	return true
}

// printSkillPipelines prints all skill pipelines that have jobs
func printSkillPipelines(skills []*model.Pipeline) {
	for _, skill := range skills {
		if hasJobs(skill) {
			printSkillSection(skill)
		}
	}
}

// hasJobs returns true if a pipeline has jobs or tasks.
func hasJobs(p *model.Pipeline) bool {
	jobs := p.Jobs
	if len(jobs) == 0 {
		jobs = p.Tasks
	}
	return len(jobs) > 0
}

// getJobs returns jobs from a pipeline, falling back to tasks if empty.
func getJobs(p *model.Pipeline) map[string]*model.Job {
	if len(p.Jobs) > 0 {
		return p.Jobs
	}
	return p.Tasks
}

// aliasEntry represents a single alias mapping.
type aliasEntry struct {
	alias  string
	target string
}

// printAliasesSection prints all aliases from skill pipelines.
// Adds spacing before section only if main pipeline was already printed.
func printAliasesSection(skills []*model.Pipeline, spaceBefore bool) {
	aliases := collectAliases(skills)
	if len(aliases) == 0 {
		return
	}

	sortAndFormatAliases(aliases)

	if spaceBefore {
		fmt.Println()
	}
	fmt.Printf("%s\n\n", colors.BrightWhite("Aliases"))
	printAliases(aliases)
}

// collectAliases gathers all aliases from skill pipelines.
func collectAliases(skills []*model.Pipeline) []aliasEntry {
	var aliases []aliasEntry

	for _, p := range skills {
		jobs := getJobs(p)

		// Skill ID alone is an alias to skill:default if default job exists
		if _, hasDefault := jobs["default"]; hasDefault {
			aliases = append(aliases, aliasEntry{
				alias:  p.ID,
				target: p.ID + ":default",
			})
		}

		// Collect explicit aliases from all jobs
		for jobName, job := range jobs {
			for _, alias := range job.Aliases {
				target := buildTargetName(p.ID, jobName)
				aliases = append(aliases, aliasEntry{
					alias:  alias,
					target: target,
				})
			}
		}
	}

	return aliases
}

// buildTargetName creates the full target reference (e.g., "go:test").
func buildTargetName(skillID, jobName string) string {
	if jobName == "default" {
		return skillID
	}
	return skillID + ":" + jobName
}

// sortAndFormatAliases sorts aliases and calculates padding for alignment.
func sortAndFormatAliases(aliases []aliasEntry) {
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].alias < aliases[j].alias
	})
}

// printAliases outputs the formatted alias list with alignment.
func printAliases(aliases []aliasEntry) {
	maxLen := calculateMaxAliasLength(aliases)

	for _, a := range aliases {
		padding := maxLen - len(a.alias) + 2
		fmt.Printf("* %s:%*s(invokes: %s)\n",
			colors.BrightGreen(a.alias),
			padding, "",
			colors.BrightOrange(a.target))
	}
}

// calculateMaxAliasLength finds the longest alias name for alignment.
func calculateMaxAliasLength(aliases []aliasEntry) int {
	maxLen := 0
	for _, a := range aliases {
		if len(a.alias) > maxLen {
			maxLen = len(a.alias)
		}
	}
	return maxLen
}

// printPipelineSection prints a pipeline with its jobs.
func printPipelineSection(p *model.Pipeline) {
	isMain := p.ID == ""
	jobs := getJobs(p)

	fmt.Printf("%s\n\n", colors.BrightWhite(p.Name))
	printJobList(jobs, p.ID, isMain)
}

// printSkillSection prints a skill pipeline section.
func printSkillSection(p *model.Pipeline) {
	fmt.Println()
	fmt.Printf("%s\n\n", colors.BrightWhite(p.Name))
	printJobList(getJobs(p), p.ID, false)
}

// printJobList outputs a formatted list of jobs with descriptions and dependencies.
func printJobList(jobs map[string]*model.Job, prefix string, isMain bool) {
	names := getSortedJobNames(jobs)
	displayNames := buildDisplayNames(names, prefix)
	maxLen := calculateMaxNameLength(displayNames)

	for i, jobName := range names {
		printJobLine(jobs[jobName], displayNames[i], maxLen, isMain)
	}
}

// getSortedJobNames returns job names sorted by depth, with "default" first.
func getSortedJobNames(jobs map[string]*model.Job) []string {
	names := treeview.SortJobsByDepth(jobNames(jobs))

	// Move "default" to the front if present
	for i, name := range names {
		if name == "default" {
			names = append([]string{name}, append(names[:i], names[i+1:]...)...)
			break
		}
	}

	return names
}

// buildDisplayNames creates display strings for job names (with optional prefix).
func buildDisplayNames(jobNames []string, prefix string) []string {
	displayNames := make([]string, len(jobNames))
	for i, name := range jobNames {
		if prefix != "" {
			displayNames[i] = prefix + ":" + name
		} else {
			displayNames[i] = name
		}
	}
	return displayNames
}

// calculateMaxNameLength finds the longest display name for alignment.
func calculateMaxNameLength(displayNames []string) int {
	maxLen := 0
	for _, name := range displayNames {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	return maxLen
}

// printJobLine outputs a single job with its metadata.
func printJobLine(job *model.Job, displayName string, maxLen int, isMain bool) {
	padding := maxLen - len(displayName) + 2
	coloredName := getJobColor(displayName, isMain)
	depsStr := buildDependsOnSuffix(job)
	aliasStr := buildAliasesSuffix(job, isMain)

	if job.Desc != "" {
		fmt.Printf("* %s:%*s%s%s%s\n", coloredName, padding, "", job.Desc, depsStr, aliasStr)
	} else if depsStr != "" || aliasStr != "" {
		fmt.Printf("* %s:%*s%s%s\n", coloredName, padding, "", depsStr, aliasStr)
	} else {
		fmt.Printf("* %s:\n", coloredName)
	}
}

// getJobColor returns the color for a job name based on pipeline type.
func getJobColor(name string, isMain bool) string {
	if isMain {
		return colors.BrightGreen(name)
	}
	return colors.BrightOrange(name)
}

// buildDependsOnSuffix creates the "(depends_on: ...)" suffix if present.
func buildDependsOnSuffix(job *model.Job) string {
	deps := GetDependencies(job.DependsOn)
	if len(deps) == 0 {
		return ""
	}

	depItems := make([]string, len(deps))
	for i, dep := range deps {
		depItems[i] = colors.BrightOrange(dep)
	}
	return fmt.Sprintf(" (depends_on: %s)", strings.Join(depItems, ", "))
}

// buildAliasesSuffix creates the "(aliases: ...)" suffix if present (main pipeline only).
func buildAliasesSuffix(job *model.Job, isMain bool) string {
	if !isMain || len(job.Aliases) == 0 {
		return ""
	}

	aliasItems := make([]string, len(job.Aliases))
	for i, alias := range job.Aliases {
		aliasItems[i] = colors.BrightGreen(alias)
	}
	return fmt.Sprintf(" (aliases: %s)", strings.Join(aliasItems, ", "))
}

// jobNames returns the keys of a job map.
func jobNames(jobs map[string]*model.Job) []string {
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}
	return names
}
