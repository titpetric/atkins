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

// HelpRequested reports whether a command line requests the help
// document and no run.
//
// A -h or --help that is the value of a string flag is not a help
// request: `atkins -x "--help"` runs a prompt. The flag set says which
// flags take a value, so the same rules tell the two cases apart that
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

// HelpFlags returns one flag set carrying every flag atkins defines.
// Help detection has to tell which of them take a value.
func HelpFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("atkins", pflag.ContinueOnError)
	NewOptions().Bind(flags)
	return flags
}

// WriteHelp renders `atkins --help` to a writer.
//
// WriteHelp scans the skills and loads none: the document lists every
// installed skill, marking what this directory does not activate, because a
// reader asking what atkins can do is not always standing in a
// repository that answers.
func WriteHelp(w io.Writer) error {
	pipelineFlags := pflag.NewFlagSet("run", pflag.ContinueOnError)
	NewOptions().Bind(pipelineFlags)

	skills, dirs := scanSkills()

	return helpdoc.Write(w, &helpdoc.Options{
		Name:    "atkins",
		Version: "Version " + Version + ", commit " + Commit + ".",
		Commands: []helpdoc.Command{
			{Name: "run", Title: "Run jobs from a pipeline file", Default: true, Flags: pipelineFlags},
		},
		Skills:    skills,
		SkillDirs: dirs,
		Color:     isTerminal(w),
	})
}

// isTerminal reports whether a writer is a terminal. A help document on
// its way to a file or a pipe stays plain markdown.
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// scanSkills returns every skill file installed for this directory, the
// repository's own first, then the parent folders', then $HOME's, and
// the directories it read them from.
func scanSkills() ([]*runner.Skill, []string) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil
	}

	loader := &runner.SkillsLoader{
		StartDir:     cwd,
		WorkspaceDir: cwd,
	}
	if _, configDir, err := runner.DiscoverConfig(cwd); err == nil && configDir != "" {
		loader.WorkspaceDir = configDir
		loader.AddWorkspace(configDir)
		loader.AddParentWorkspaces()
	}
	if home, err := os.UserHomeDir(); err == nil {
		loader.AddSkillsDir(filepath.Join(home, ".atkins", "skills"))
	}

	skills, err := loader.Scan()
	if err != nil {
		return nil, loader.SkillsDirs
	}

	return skills, loader.SkillsDirs
}
