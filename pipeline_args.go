package main

import (
	"fmt"
	"strings"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// loadSkills loads the active skills, and with -l --all the inactive
// ones too.
func loadSkills(loader *runner.SkillsLoader, opts *Options) ([]*model.Pipeline, error) {
	if opts.List && opts.All {
		return loader.LoadAll()
	}
	return loader.Load()
}

// splitJobArgs separates the arguments after a bare -- from the job
// names and files before it. dash is the position of -- in the parsed
// arguments, as pflag's ArgsLenAtDash reports it, and parsed holds
// len(parsed)-len(args) leading arguments the command consumed.
func splitJobArgs(dash, parsed int, args []string) (names, jobArgs []string) {
	if dash < 0 {
		return args, nil
	}
	dash -= parsed - len(args)
	if dash < 0 {
		dash = 0
	}
	if dash > len(args) {
		dash = len(args)
	}
	return args[:dash], args[dash:]
}

// checkJobArgs refuses arguments after -- unless one job runs and it
// declares args: true.
func checkJobArgs(jobArgs []string, jobs []string, targets []*model.ResolvedTask) error {
	if len(jobArgs) == 0 {
		return nil
	}
	if len(targets) != 1 {
		return fmt.Errorf("arguments after -- need exactly one job, got %d: %s", len(jobs), strings.Join(jobs, " "))
	}
	if !targets[0].Job.Args {
		return fmt.Errorf("job %q takes no arguments; it needs args: true", targets[0].Name)
	}
	return nil
}
