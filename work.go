package main

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/titpetric/cli"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/worker"
)

// WorkerOptions holds `atkins worker` command-line arguments.
//
// Flags override the resolved configuration, which already carries the
// document values and the ATKINS_* overlay. An empty flag leaves the
// configured value alone, so the three layers stack rather than fight.
type WorkerOptions struct {
	Server   string
	Token    string
	Email    string
	Password string
	AgentID  string
	Labels   string
	DataDir  string
}

// Bind registers the worker flags.
func (o *WorkerOptions) Bind(fs *pflag.FlagSet) {
	fs.StringVar(&o.Server, "server", "", "Server to claim jobs from (default: configured, or the logged-in server)")
	fs.StringVar(&o.Token, "token", "", "Shared enrolment token from the server")
	fs.StringVar(&o.Email, "email", "", "Agent account email, when enrolling with a token isn't used")
	fs.StringVar(&o.Password, "password", "", "Agent account password")
	fs.StringVar(&o.AgentID, "agent-id", "", "Agent identity in job leases (default: hostname)")
	fs.StringVar(&o.Labels, "labels", "", "Comma separated labels this agent advertises")
	fs.StringVar(&o.DataDir, "data-dir", "", "Directory for the repository cache and job work trees")
}

// Worker provides a cli.Command that runs the atkins CI/CD agent.
func Worker() *cli.Command {
	opts := &WorkerOptions{}

	return &cli.Command{
		Name:  "worker",
		Title: "Claim and run CI/CD jobs from a server",
		Bind:  opts.Bind,
		Run: func(ctx context.Context, _ []string) error {
			return runWorker(ctx, opts)
		},
	}
}

func runWorker(ctx context.Context, opts *WorkerOptions) error {
	cwd, _ := os.Getwd()

	settings, err := loadRuntimeConfig(cwd)
	if err != nil {
		return err
	}

	workerOpts := worker.FromConfig(settings)
	override(&workerOpts.Server, opts.Server)
	override(&workerOpts.Token, opts.Token)
	override(&workerOpts.Email, opts.Email)
	override(&workerOpts.Password, opts.Password)
	override(&workerOpts.AgentID, opts.AgentID)
	override(&workerOpts.DataDir, opts.DataDir)

	if strings.TrimSpace(opts.Labels) != "" {
		workerOpts.Labels = config.SplitList(opts.Labels)
	}

	agent, err := worker.New(ctx, workerOpts)
	if err != nil {
		return err
	}

	return agent.Run(ctx)
}

// override replaces a configured value when a flag was given.
func override(target *string, value string) {
	if strings.TrimSpace(value) != "" {
		*target = value
	}
}
