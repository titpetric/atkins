package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/titpetric/cli"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
	"github.com/titpetric/atkins/storage"
	"github.com/titpetric/atkins/version"
)

// loadSkillPipelines loads skill pipelines from the project-local .atkins/skills/ directory,
// then from the .atkins/skills/ of every parent folder unless jailed.
// workspaceDir is the folder containing .atkins/ (used as Dir for skills without when:).
// startDir is where to start searching for when: files (typically user's cwd).
func loadSkillPipelines(workspaceDir string, startDir string, opts *Options) ([]*model.Pipeline, error) {
	loader := runner.NewSkillsLoader(workspaceDir, startDir)
	if !opts.Jail {
		loader.AddParentWorkspaces()
	}
	return loadSkills(loader, opts)
}

// pipelineJobs is one pipeline and the jobs resolved onto it.
type pipelineJobs struct {
	pipeline *model.Pipeline
	jobs     []string
}

// stdinHasData checks if stdin has data available without blocking.
// Returns true if stdin comes from a pipe or redirect with data available.
func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is not a terminal (i.e., a pipe or a redirect)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return false
	}
	// For regular files, check size. For pipes, size is 0 but data may be available.
	// Only treat as having data if it's a regular file with content, or a named pipe.
	// This avoids blocking on empty pipes (e.g., from `timeout` command).
	mode := stat.Mode()
	if mode.IsRegular() {
		return stat.Size() > 0
	}
	// For named pipes (FIFOs), report data as available
	if mode&os.ModeNamedPipe != 0 {
		return true
	}
	// For anonymous pipes, we can't easily check without blocking.
	// Check if it looks like a pipe from a shell command (mode is irregular)
	// by checking if we're NOT a socket, device, etc.
	if mode&(os.ModeSocket|os.ModeDevice) == 0 {
		return true
	}
	return false
}

// Pipeline returns a cli.Command that runs the atkins command pipeline.
func Pipeline() *cli.Command {
	opts := NewOptions()

	return &cli.Command{
		Name:    "_pipeline",
		Title:   "Pipeline automation tool",
		Default: true,
		Bind: func(fs *pflag.FlagSet) {
			opts.Bind(fs)
		},
		Run: func(ctx context.Context, args []string) error {
			return runPipeline(ctx, opts, args)
		},
	}
}

func runPipeline(ctx context.Context, opts *Options, args []string) error {
	// Handle version flag early, before any file discovery
	if opts.Version {
		return version.Run(os.Stdout, version.Info{
			Version:    Version,
			Commit:     Commit,
			CommitTime: CommitTime,
			Branch:     Branch,
		})
	}

	cwd, _ := os.Getwd()

	// Configuration resolves before anything reads a setting, so a
	// broken document surfaces once and by name.
	settings, err := loadRuntimeConfig(cwd)
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}

	// These flags take over the whole invocation: they open the
	// configuration menu, vendor skills into the repository or report
	// the recorded runs, and exit.
	switch {
	case opts.Config:
		return runConfig(cwd)
	case opts.Vendor:
		return runVendor(cwd, opts)
	case opts.Stats:
		return runStats(ctx, settings, opts)
	}

	// Check mutually exclusive flags
	if opts.JSON && opts.YAML {
		return fmt.Errorf("%s --json and --yaml flags cannot be combined", colors.BrightRed("ERROR:"))
	}

	fileFlag := opts.FlagSet.Lookup("file")

	// Everything after a bare -- goes to the job as arguments, and is
	// never a job name or a pipeline file.
	if opts.FlagSet != nil {
		args, opts.Args = splitJobArgs(opts.FlagSet.ArgsLenAtDash(), len(opts.FlagSet.Args()), args)
	}

	// Handle positional arguments before changing directory
	fileExplicitlySet := fileFlag != nil && fileFlag.Changed
	for _, arg := range args {
		// Check if arg is an existing regular file (shebang invocation)
		if info, err := os.Stat(arg); err == nil && info.Mode().IsRegular() {
			opts.File = arg
			fileExplicitlySet = true
			continue
		}

		// Collect all job names from positional arguments
		opts.Jobs = append(opts.Jobs, arg)
	}

	// Save original working directory for global skill when.files checks,
	// since cwd may change during configuration/environment discovery.
	originalCwd, _ := os.Getwd()

	// Check stdin first (before file discovery)
	var pipelines []*model.Pipeline

	if stdinHasData() {
		// Read pipeline from stdin
		pipelines, err = runner.LoadPipelineFromReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("%s %s", colors.BrightRed("ERROR:"), err)
		}
		// Set default name if not specified
		if pipelines[0].Name == "" {
			pipelines[0].Name = "stdin"
		}
		opts.File = "stdin"
	} else {
		// Discover or resolve pipeline file before changing directory
		var absPath string
		var configDir string

		if fileExplicitlySet {
			// If the user passed -f/--file, use it directly. Its folder
			// is where the skills load from, as for a discovered file;
			// atkins stays in the working directory.
			absPath, err = filepath.Abs(opts.File)
			if err != nil {
				return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
			}
			configDir = filepath.Dir(absPath)
		} else {
			// Discover the configuration file by traversing parent directories
			var configPath string
			var discoverErr error
			configPath, configDir, discoverErr = runner.DiscoverConfigFromCwd()
			if discoverErr != nil && configPath != "" {
				// No configuration file found: try environment autodiscovery
				env, envErr := runner.DiscoverEnvironmentFromCwd()
				if envErr != nil {
					// Neither configuration nor environment found
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), discoverErr)
				}

				// Change to the discovered project root
				if err := os.Chdir(env.Root); err != nil {
					return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), env.Root, err)
				}

				// Load and merge skill pipelines
				pipelines, err = loadSkillPipelines(env.Root, originalCwd, opts)
				if err != nil {
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
				}
				opts.File = "<autodiscovered>"
				goto pipelineReady
			}
			absPath = configPath
			opts.File = configPath

			// Only change directory when discovery finds a configuration file (not just an .atkins/ folder).
			// For skills-only mode, stay in user's working directory.
			if configPath != "" {
				if err := os.Chdir(configDir); err != nil {
					return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), configDir, err)
				}
			}
		}

		// Load and parse pipeline from detected configuration path.
		// An .atkins folder here leaves the configuration path empty.
		if absPath != "" {
			pipelines, err = runner.LoadPipeline(absPath)
			if err != nil {
				return fmt.Errorf("%s %s", colors.BrightRed("ERROR:"), err)
			}

			// Merge autodiscovered skills into the loaded pipeline
			if skillPipelines, skillErr := loadSkillPipelines(configDir, originalCwd, opts); skillErr == nil {
				pipelines = append(pipelines, skillPipelines...)
			}
		} else {
			// .atkins/ folder detected without a configuration file - load skills as primary pipelines
			opts.File = ".atkins/"
			if skillPipelines, skillErr := loadSkillPipelines(configDir, originalCwd, opts); skillErr == nil {
				pipelines = skillPipelines
			}
		}
	}

pipelineReady:

	// Always merge global skills from $HOME/.atkins/skills/ (unless jailed).
	// Local .atkins/skills/ takes precedence: skip globals already loaded by ID.
	if !opts.Jail {
		if home, err := os.UserHomeDir(); err == nil {
			globalLoader := runner.NewSkillsLoader(originalCwd, originalCwd)
			globalLoader.SkillsDirs = []string{filepath.Join(home, ".atkins", "skills")}
			if globalPipelines, globalErr := loadSkills(globalLoader, opts); globalErr == nil {
				pipelines = runner.MergeSkills(pipelines, globalPipelines)
			}
		}
	}

	// A job's own when: block gates it on the folder atkins runs from.
	runner.ApplyJobWhen(pipelines, originalCwd)

	// Handle working directory override (applies to both stdin and file modes)
	if opts.WorkingDirectory != "" {
		if err := os.Chdir(opts.WorkingDirectory); err != nil {
			return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), opts.WorkingDirectory, err)
		}
	}

	// Handle lint mode
	if opts.Lint || opts.List {
		if done, err := lintPipelines(opts, pipelines); done || err != nil {
			return err
		}
	}

	// Save all pipelines for cross-pipeline task references
	allPipelines := pipelines

	// Handle list mode
	if opts.List {
		// If the user names a job/skill, filter to that pipeline
		if len(opts.Jobs) > 0 {
			for _, p := range pipelines {
				if p.ID == opts.Jobs[0] {
					pipelines = []*model.Pipeline{p}
					break
				}
			}
		}

		listOpts := runner.OutputOptions{All: opts.All}
		listOpts.Dir, _ = os.Getwd()

		if opts.JSON {
			return runner.ListPipelinesJSON(pipelines, listOpts)
		}
		if opts.YAML {
			return runner.ListPipelinesYAML(pipelines, listOpts)
		}

		fmt.Print(runner.ListPipelines(runner.ActivePipelines(pipelines)))
		return nil
	}

	// When no jobs specified, run the default job from main pipeline
	if len(opts.Jobs) == 0 {
		opts.Jobs = []string{"default"}
	}

	// Resolve all jobs and group by pipeline
	pipelineJobsMap := make(map[*model.Pipeline]*pipelineJobs)
	var pipelineOrder []*model.Pipeline
	resolver := runner.NewTaskResolver(pipelines)

	// For implicit "default", resolve against the primary pipeline only
	// to avoid matching skill defaults when the main pipeline has none.
	if len(pipelines) > 0 {
		primaryResolver := runner.NewTaskResolver(pipelines[:1])
		if target, err := primaryResolver.Resolve("default"); err == nil {
			_ = target // primary has default, resolver will find it
		} else if len(opts.Jobs) == 1 && opts.Jobs[0] == "default" {
			resolver = primaryResolver
		}
	}

	var targets []*model.ResolvedTask
	for _, jobName := range opts.Jobs {
		target, err := resolver.Resolve(jobName)
		if err != nil {
			// Check if it's a fuzzy match error with more than one match
			var fuzzyErr *runner.FuzzyMatchError
			if errors.As(err, &fuzzyErr) {
				fmt.Fprintf(os.Stderr, "%s found %d matching jobs:\n\n", colors.BrightYellow("INFO:"), len(fuzzyErr.Matches))
				for _, match := range fuzzyErr.Matches {
					displayName := match.Name
					if match.Pipeline.ID != "" {
						displayName = match.Pipeline.ID + ":" + match.Name
					}
					fmt.Fprintf(os.Stderr, "  - %s\n", colors.BrightOrange(displayName))
				}
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "%s %v\n", colors.BrightRed("ERROR:"), err)
			fmt.Fprintf(os.Stderr, "\nUsage: atkins [flags] [job-names...]\n")
			os.Exit(1)
		}

		if target.Job.Inactive {
			fmt.Fprintf(os.Stderr, "%s job %q does not apply here: its when: block does not match\n", colors.BrightRed("ERROR:"), target.Name)
			os.Exit(1)
		}
		targets = append(targets, target)

		pipeline := target.Pipeline
		if pipelineJobsMap[pipeline] == nil {
			pipelineJobsMap[pipeline] = &pipelineJobs{pipeline: pipeline}
			pipelineOrder = append(pipelineOrder, pipeline)
		}
		// Strip pipeline ID prefix since RunPipeline uses raw job map keys
		resolvedName := target.Name
		if pipeline.ID != "" {
			resolvedName = strings.TrimPrefix(resolvedName, pipeline.ID+":")
		}
		pipelineJobsMap[pipeline].jobs = append(pipelineJobsMap[pipeline].jobs, resolvedName)
	}

	if err := checkJobArgs(opts.Args, opts.Jobs, targets); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", colors.BrightRed("ERROR:"), err)
		os.Exit(1)
	}

	// The event database records what ran and how long it took, so
	// `atkins --stats` and task-ui can report on it. When no database
	// opens, the recorder stays nil and the pipeline runs unrecorded.
	recorder := openRecorder(settings, opts)

	exitCode, _ := runPipelines(ctx, opts, pipelineOrder, pipelineJobsMap, allPipelines, recorder.recorder)

	// Closed here and not deferred: the exit below skips every defer,
	// and a sqlite file left open keeps its write-ahead log unmerged.
	recorder.Close()

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

// runPipelines runs each pipeline with its collected jobs and returns
// the exit code the shell should see.
//
// The caller performs the exit, and this function never does: the
// recorder writes its rows as each pipeline finishes, and os.Exit from
// here would file none of it.
func runPipelines(
	ctx context.Context,
	opts *Options,
	pipelineOrder []*model.Pipeline,
	pipelineJobsMap map[*model.Pipeline]*pipelineJobs,
	allPipelines []*model.Pipeline,
	recorder runner.Recorder,
) (int, error) {
	exitCode := 0
	var runErr error

	for _, pipeline := range pipelineOrder {
		pj := pipelineJobsMap[pipeline]
		err := runner.RunPipeline(ctx, pipeline, runner.PipelineOptions{
			Jobs:         pj.jobs,
			LogFile:      opts.LogFile,
			PipelineFile: opts.File,
			Debug:        opts.Debug,
			FinalOnly:    opts.FinalOnly,
			JSON:         opts.JSON,
			YAML:         opts.YAML,
			AllPipelines: allPipelines,
			Recorder:     recorder,
			Args:         opts.Args,
		})
		if err == nil {
			continue
		}

		runErr = err
		exitCode = 1
		failedPipeline := pipeline.Name

		var errorLog runner.ExecError
		if errors.As(err, &errorLog) {
			if errorLog.Len() > 0 {
				fmt.Fprintf(os.Stderr, "\nAn error occurred in %q pipeline:\n\n", failedPipeline)
				fmt.Fprintf(os.Stderr, "  Exit code: %d\n", errorLog.LastExitCode)
				fmt.Fprintf(os.Stderr, "  Error output:\n")
				for _, line := range strings.Split(errorLog.Output, "\n") {
					if line != "" {
						fmt.Fprintf(os.Stderr, "    %s\n", line)
					}
				}
			}
			exitCode = errorLog.LastExitCode
		} else {
			fmt.Fprintf(os.Stderr, "\nAn error occurred in %q pipeline:\n", failedPipeline)
			fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
		}

		// An exit code of zero is a failure the pipeline reported
		// without one, and the remaining pipelines still run; that is
		// the behaviour every earlier release had.
		if exitCode != 0 {
			return exitCode, runErr
		}
	}

	return exitCode, runErr
}

// recording is the event databases for one invocation, and the recorder
// over them. Both are empty when nothing is recording, and Close
// tolerates that, so the caller has one branch and not three.
type recording struct {
	stores   []*storage.Storage
	recorder runner.Recorder
}

// Close releases the databases.
func (r recording) Close() {
	for _, store := range r.stores {
		_ = store.Close()
	}
}

// openRecorder returns an empty recording for `--local`, `record: false` and
// when no database opens; it prints the error and the pipeline still runs.
// It returns a recording over every database that opens otherwise.
func openRecorder(settings *config.Config, opts *Options) recording {
	if opts.Local || !settings.Storage.Record {
		return recording{}
	}

	stores, err := openStorage(settings)
	if err != nil {
		what := "not recording this run"
		if len(stores) > 0 {
			what = "recording this run to " + storagePaths(stores) + " alone"
		}
		fmt.Fprintf(os.Stderr, "%s %s: %v\n", colors.BrightYellow("INFO:"), what, err)
	}
	if len(stores) == 0 {
		return recording{}
	}

	return recording{stores: stores, recorder: storage.NewRecorders(stores)}
}

// openStorage opens the databases a run records to: the project's, the
// workspace's, which the configuration names in place of $HOME's, and the
// mirrors named after it.
func openStorage(settings *config.Config) ([]*storage.Storage, error) {
	return storage.NewStorage(settings.Storage.Database)
}

// storagePaths names the open databases, comma separated.
func storagePaths(stores []*storage.Storage) string {
	names := make([]string, 0, len(stores))
	for _, store := range stores {
		names = append(names, store.Path())
	}
	return strings.Join(names, ", ")
}
