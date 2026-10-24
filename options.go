package main

import (
	"github.com/titpetric/cli"
)

// Options holds pipeline command-line arguments
type Options struct {
	File             string
	Jobs             []string
	Args             []string
	List             bool
	All              bool
	Lint             bool
	Debug            bool
	LogFile          string
	FinalOnly        bool
	WorkingDirectory string
	Jail             bool
	JSON             bool
	YAML             bool
	Version          bool

	// Stats reports the commands recorded in the event database by the
	// time they have taken, and runs no pipeline. Local records the run
	// nowhere.
	Stats bool
	Local bool

	// Limit keeps the newest commands --stats --all lists. Zero keeps
	// every command of the window.
	Limit int

	// Config opens the configuration menu for .atkins/config.yml.
	Config bool

	// Vendor copies the skills this repository uses into .atkins/skills.
	// It reports the selection and writes nothing unless Write is true.
	Vendor bool
	Write  bool

	// Prune removes the runs and commands older than OlderThan from the
	// database DB names and gives the space back, and runs no pipeline;
	// DryRun counts them and changes nothing.
	Prune     bool
	DB        string
	OlderThan string
	DryRun    bool

	FlagSet *cli.FlagSet
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) Bind(fs *cli.FlagSet) {
	fs.StringVarP(&o.File, "file", "f", "", "Path to pipeline file (auto-discovers .atkins.yml)")
	fs.BoolVarP(&o.List, "list", "l", false, "List pipeline jobs and dependencies")
	fs.BoolVar(&o.All, "all", false, "With --list, include the skills whose when: block does not match; with --stats, list every recorded command")
	fs.BoolVar(&o.Lint, "lint", false, "Lint pipeline for errors")
	fs.BoolVar(&o.Debug, "debug", false, "Print debug data")
	fs.StringVar(&o.LogFile, "log", "", "Log file path for command execution")
	fs.BoolVar(&o.FinalOnly, "final", false, "Only render final output without redrawing (no interactive tree)")
	fs.StringVarP(&o.WorkingDirectory, "working-directory", "w", "", "Change to this directory before running")
	fs.BoolVar(&o.Jail, "jail", false, "Restrict to project scope, skip skills from parent folders and global resources from $HOME")
	fs.BoolVarP(&o.JSON, "json", "j", false, "Output in JSON format")
	fs.BoolVarP(&o.YAML, "yaml", "y", false, "Output in YAML format")
	fs.BoolVarP(&o.Version, "version", "v", false, "Print version and build information")
	fs.BoolVar(&o.Stats, "stats", false, "Report the recorded commands by the time they have taken, and run nothing")
	fs.IntVar(&o.Limit, "limit", 0, "With --stats --all, list only the newest N commands (0 lists them all)")
	fs.BoolVar(&o.Local, "local", false, "Run without recording anything in the event database")
	fs.BoolVar(&o.Config, "config", false, "Open the configuration menu for .atkins/config.yml")
	fs.BoolVar(&o.Vendor, "vendor", false, "List the skills this repository uses from $HOME/.atkins/skills")
	fs.BoolVar(&o.Write, "write", false, "With --vendor, write the skills into .atkins/skills")
	fs.BoolVar(&o.Prune, "prune", false, "Remove the runs and commands older than --older-than from the database --db names, give the space back, and run nothing")
	fs.StringVar(&o.DB, "db", "", "With --prune, the database file to prune")
	fs.StringVar(&o.OlderThan, "older-than", "31d", "With --prune, the age past which a run goes: days, such as 31d, or a duration, such as 720h")
	fs.BoolVar(&o.DryRun, "dry-run", false, "With --prune, count what would go and change nothing")

	o.FlagSet = fs
}
