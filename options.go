package main

import "github.com/titpetric/cli"

// Options holds pipeline command-line arguments
type Options struct {
	File             string
	Job              string
	List             bool
	Lint             bool
	Debug            bool
	LogFile          string
	FinalOnly        bool
	WorkingDirectory string
	Jail             bool
	JSON             bool
	YAML             bool

	FlagSet *cli.FlagSet
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) Bind(fs *cli.FlagSet) {
	fs.StringVarP(&o.File, "file", "f", "", "Path to pipeline file (auto-discovers .atkins.yml)")
	fs.StringVar(&o.Job, "job", "", "Specific job to run")
	fs.BoolVarP(&o.List, "list", "l", false, "List pipeline jobs and dependencies")
	fs.BoolVar(&o.Lint, "lint", false, "Lint pipeline for errors")
	fs.BoolVar(&o.Debug, "debug", false, "Print debug data")
	fs.StringVar(&o.LogFile, "log", "", "Log file path for command execution")
	fs.BoolVar(&o.FinalOnly, "final", false, "Only render final output without redrawing (no interactive tree)")
	fs.StringVarP(&o.WorkingDirectory, "working-directory", "w", "", "Change to this directory before running")
	fs.BoolVar(&o.Jail, "jail", false, "Restrict to project scope, skip global resources from $HOME")
	fs.BoolVarP(&o.JSON, "json", "j", false, "Output in JSON format")
	fs.BoolVarP(&o.YAML, "yaml", "y", false, "Output in YAML format")

	o.FlagSet = fs
}
