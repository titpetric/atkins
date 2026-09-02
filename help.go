package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/term"

	"github.com/titpetric/atkins/helpdoc"
	"github.com/titpetric/atkins/runner"
)

// HelpRequested reports whether a command line is asking for the help
// document rather than a run.
//
// A -h or --help that is the value of a string flag is not a help
// request: `atkins -x "--help"` runs a prompt. The flag set says which
// flags take a value, so the two cases are told apart by the same rules
// pflag parses by. Everything after a bare -- is an argument.
func HelpRequested(flags *pflag.FlagSet, args []string) bool {
	for index := 0; index < len(args); index++ {
		arg := args[index]

		switch {
		case arg == "--":
			return false
		case arg == "-h", arg == "--help":
			return true
		case takesValue(flags, arg):
			index++
		}
	}

	return false
}

// takesValue reports whether an argument names a flag that consumes the
// argument after it. A flag written as --name=value carries its own.
func takesValue(flags *pflag.FlagSet, arg string) bool {
	if flags == nil || len(arg) < 2 || arg[0] != '-' {
		return false
	}
	if strings.Contains(arg, "=") {
		return false
	}

	var flag *pflag.Flag
	switch {
	case len(arg) > 2 && arg[1] == '-':
		flag = flags.Lookup(arg[2:])
	case arg[1] != '-':
		flag = flags.ShorthandLookup(arg[1:])
	}

	return flag != nil && flag.Value.Type() != "bool"
}

// HelpFlags returns one flag set carrying every flag atkins defines, for
// the three commands together. Help detection needs to know which of
// them take a value, and the three sets share no names.
func HelpFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("atkins", pflag.ContinueOnError)
	NewOptions().Bind(flags)
	(&ServerOptions{}).Bind(flags)
	(&WorkerOptions{}).Bind(flags)
	return flags
}

// WriteHelp renders `atkins --help` to a writer.
//
// The skills are scanned rather than loaded: the document lists what is
// installed, marking what this directory does not activate, because a
// reader asking what atkins can do is not always standing in a
// repository that answers.
func WriteHelp(w io.Writer) error {
	pipelineFlags := pflag.NewFlagSet("run", pflag.ContinueOnError)
	NewOptions().Bind(pipelineFlags)

	serverFlags := pflag.NewFlagSet("server", pflag.ContinueOnError)
	(&ServerOptions{}).Bind(serverFlags)

	workerFlags := pflag.NewFlagSet("worker", pflag.ContinueOnError)
	(&WorkerOptions{}).Bind(workerFlags)

	skills, dirs := scanSkills()

	return helpdoc.Write(w, &helpdoc.Options{
		Name:    "atkins",
		Version: "Version " + Version + ", commit " + Commit + ".",
		Commands: []helpdoc.Command{
			{Name: "run", Title: "Run jobs from a pipeline file", Default: true, Flags: pipelineFlags},
			{Name: "server", Title: "Run the CI/CD server", Flags: serverFlags},
			{Name: "worker", Title: "Run the CI/CD agent, claiming jobs from a server", Flags: workerFlags},
		},
		Skills:    skills,
		SkillDirs: dirs,
		Color:     isTerminal(w),
	})
}

// isTerminal reports whether a writer is a terminal. A help document on
// its way to a file or a pipe is left as plain markdown.
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// scanSkills returns every skill file installed for this directory, the
// repository's own first, and the directories they were read from.
func scanSkills() ([]*runner.Skill, []string) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil
	}

	var dirs []string
	if _, configDir, err := runner.DiscoverConfig(cwd); err == nil && configDir != "" {
		dirs = append(dirs, filepath.Join(configDir, ".atkins", "skills"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".atkins", "skills"))
	}

	loader := &runner.SkillsLoader{
		SkillsDirs:   dirs,
		StartDir:     cwd,
		WorkspaceDir: cwd,
	}

	skills, err := loader.Scan()
	if err != nil {
		return nil, dirs
	}

	return skills, dirs
}
