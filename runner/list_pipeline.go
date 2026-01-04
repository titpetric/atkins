package runner

import (
	"fmt"

	"github.com/titpetric/atkins-ci/model"
	"github.com/titpetric/atkins-ci/treeview"
)

// ListPipeline displays a pipeline's job tree with dependencies
func ListPipeline(pipeline *model.Pipeline) {
	allJobs := pipeline.Jobs
	if len(allJobs) == 0 {
		allJobs = pipeline.Tasks
	}

	// Get jobs in dependency order
	jobOrder, err := ResolveJobDependencies(allJobs, "")
	if err != nil {
		fmt.Printf("%s %s\n", "ERROR:", err)
		return
	}

	// Build tree using treeview builder
	builder := treeview.NewBuilder(pipeline.Name)

	for _, jobName := range jobOrder {
		job := allJobs[jobName]

		// Get dependencies
		deps := GetDependencies(job.DependsOn)

		// Add job to tree
		builder.AddJob(jobName, job, deps)
	}

	// Render the tree
	display := treeview.NewDisplay()
	display.RenderStatic(builder.Root())
}
