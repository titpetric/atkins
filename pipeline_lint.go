package main

import (
	"fmt"
	"io"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// lintPipelines lints the pipelines for --lint and --list. It reports
// done when the invocation ends here: --lint has nothing more to do, and
// a lint failure ends it with io.EOF after printing the findings, as
// JSON or YAML with --json or --yaml.
func lintPipelines(opts *Options, pipelines []*model.Pipeline) (done bool, err error) {
	lintable := activeSkills(pipelines)
	for _, pipeline := range lintable {
		linter := runner.NewLinterWithPipelines(pipeline, lintable)
		lintErrors := linter.Lint()
		if len(lintErrors) == 0 {
			continue
		}

		switch {
		case opts.JSON:
			_ = runner.PrintJSON(runner.NewOutputError(pipeline, lintErrors))
		case opts.YAML:
			_ = runner.PrintYAML(runner.NewOutputError(pipeline, lintErrors))
		default:
			fmt.Printf("%s Pipeline '%s' has errors:\n", colors.BrightRed("✗"), pipeline.Name)
			for _, lintErr := range lintErrors {
				fmt.Printf("  %s: %s\n", lintErr.Job, lintErr.Detail)
			}
		}
		return true, io.EOF
	}

	if opts.Lint {
		if len(pipelines) > 0 {
			fmt.Printf("%s Pipeline '%s' is valid\n", colors.BrightGreen("✓"), pipelines[0].Name)
		}
		return true, nil
	}
	return false, nil
}

// activeSkills returns the pipelines without the inactive skills a
// listing with --all carries.
func activeSkills(pipelines []*model.Pipeline) []*model.Pipeline {
	result := make([]*model.Pipeline, 0, len(pipelines))
	for _, p := range pipelines {
		if !p.Inactive {
			result = append(result, p)
		}
	}
	return result
}
