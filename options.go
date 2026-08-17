package main

import "github.com/titpetric/cli"

// Options holds pipeline command-line arguments
type Options struct {
	File             string
	Jobs             []string
	List             bool
	Lint             bool
	Debug            bool
	LogFile          string
	FinalOnly        bool
	WorkingDirectory string
	Jail             bool
	JSON             bool
	YAML             bool
	Version          bool
	Agent            bool
	Exec             string

	// CI/CD client flags. Login and Register take the server URL;
	// Logout applies to the server last logged in to. Dispatch hands
	// the run to an agent instead of running it here, and Local runs
	// here without recording anything on the server.
	Login    string
	Register string
	Logout   bool
	Dispatch bool
	Local    bool

	// Config opens the configuration menu for .atkins/config.yml.
	Config bool

	// Vendor copies the skills this repository uses into .atkins/skills.
	// It reports the selection and writes nothing unless Write is set.
	Vendor bool
	Write  bool

	FlagSet *cli.FlagSet
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) Bind(fs *cli.FlagSet) {
	fs.StringVarP(&o.File, "file", "f", "", "Path to pipeline file (auto-discovers .atkins.yml)")
	fs.BoolVarP(&o.List, "list", "l", false, "List pipeline jobs and dependencies")
	fs.BoolVar(&o.Lint, "lint", false, "Lint pipeline for errors")
	fs.BoolVar(&o.Debug, "debug", false, "Print debug data")
	fs.StringVar(&o.LogFile, "log", "", "Log file path for command execution")
	fs.BoolVar(&o.FinalOnly, "final", false, "Only render final output without redrawing (no interactive tree)")
	fs.StringVarP(&o.WorkingDirectory, "working-directory", "w", "", "Change to this directory before running")
	fs.BoolVar(&o.Jail, "jail", false, "Restrict to project scope, skip global resources from $HOME")
	fs.BoolVarP(&o.JSON, "json", "j", false, "Output in JSON format")
	fs.BoolVarP(&o.YAML, "yaml", "y", false, "Output in YAML format")
	fs.BoolVarP(&o.Version, "version", "v", false, "Print version and build information")
	fs.BoolVar(&o.Agent, "agent", false, "Start interactive agent REPL")
	fs.StringVarP(&o.Exec, "exec", "x", "", "Run a prompt non-interactively and exit")
	fs.StringVar(&o.Login, "login", "", "Log in to an atkins CI/CD server, e.g. --login https://ci.example.com")
	fs.StringVar(&o.Register, "register", "", "Register an account on an atkins CI/CD server and log in")
	fs.BoolVar(&o.Logout, "logout", false, "Log out of the atkins CI/CD server")
	fs.BoolVar(&o.Dispatch, "dispatch", false, "Hand this run to a CI/CD agent instead of running it here")
	fs.BoolVar(&o.Local, "local", false, "Run here without recording the run on the CI/CD server")
	fs.BoolVar(&o.Config, "config", false, "Open the configuration menu for .atkins/config.yml")
	fs.BoolVar(&o.Vendor, "vendor", false, "List the skills this repository uses from $HOME/.atkins/skills")
	fs.BoolVar(&o.Write, "write", false, "With --vendor, write the skills into .atkins/skills")

	o.FlagSet = fs
}
