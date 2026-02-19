package runner

import (
	"fmt"
	"sort"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ListPipelines displays pipelines grouped by section in a flat list format.
// The first pipeline is the default; subsequent ones are skill pipelines.
// Skill pipeline jobs that already exist in the default pipeline are excluded.
func ListPipelines(pipelines []*model.Pipeline) {
	if len(pipelines) == 0 {
		return
	}

	// Collect job names from the default pipeline to filter skill duplicates.
	defaultJobs := make(map[string]bool)
	main := pipelines[0]
	for name := range main.Jobs {
		defaultJobs[name] = true
	}
	for name := range main.Tasks {
		defaultJobs[name] = true
	}

	printPipelineSection(main)

	// Skill pipelines sorted by name for consistent output.
	skills := make([]*model.Pipeline, len(pipelines)-1)
	copy(skills, pipelines[1:])
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	for _, p := range skills {
		printSkillSection(p, defaultJobs)
	}
}

// printPipelineSection prints a single pipeline section with all its jobs.
func printPipelineSection(p *model.Pipeline) {
	jobs := p.Jobs
	if len(jobs) == 0 {
		jobs = p.Tasks
	}

	fmt.Printf("%s\n\n", colors.BrightCyan(p.Name))
	printJobList(jobs)
}

// printSkillSection prints a skill pipeline, excluding jobs already in the default pipeline.
func printSkillSection(p *model.Pipeline, exclude map[string]bool) {
	jobs := filterJobs(p.Jobs, exclude)
	tasks := filterJobs(p.Tasks, exclude)

	// Merge tasks into jobs for display
	for k, v := range tasks {
		jobs[k] = v
	}

	if len(jobs) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("%s\n\n", colors.BrightCyan(p.Name))
	printJobList(jobs)
}

// printJobList prints a flat list of jobs with aligned descriptions.
func printJobList(jobs map[string]*model.Job) {
	names := treeview.SortJobsByDepth(jobNames(jobs))

	// Find max name length for alignment.
	maxLen := 0
	for _, name := range names {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	for _, name := range names {
		job := jobs[name]
		desc := job.Desc
		padding := maxLen - len(name) + 2

		if desc != "" {
			fmt.Printf("* %s:%*s%s\n", colors.BrightYellow(name), padding, "", desc)
		} else {
			fmt.Printf("* %s:\n", colors.BrightYellow(name))
		}
	}
}

// filterJobs returns a copy of jobs excluding any names present in the exclude set.
func filterJobs(jobs map[string]*model.Job, exclude map[string]bool) map[string]*model.Job {
	result := make(map[string]*model.Job)
	for name, job := range jobs {
		if !exclude[name] {
			result[name] = job
		}
	}
	return result
}

// jobNames returns the keys of a job map.
func jobNames(jobs map[string]*model.Job) []string {
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}
	return names
}
