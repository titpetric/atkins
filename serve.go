package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/titpetric/platform/pkg/drivers"

	"github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server"
)

// ServerOptions holds `atkins server` command-line arguments.
//
// Every flag overrides the resolved configuration; an empty one leaves
// the configured value alone.
type ServerOptions struct {
	// Addr is the listen address.
	Addr string

	// Database is the platform DSN, e.g. sqlite://file:atkins.db or
	// mysql://user:pass@tcp(host)/atkins.
	Database string

	// ArtefactDir is where the bytes of job artefacts are stored.
	ArtefactDir string

	// SigningKey signs access tokens. Required.
	SigningKey string

	// AgentToken is the shared secret agents enrol with.
	AgentToken string

	// AllowRegistration keeps /api/user/register open after the first
	// user exists.
	AllowRegistration bool
}

// Bind registers the server flags.
func (o *ServerOptions) Bind(fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, "addr", "", "Listen address")
	fs.StringVar(&o.Database, "database", "", "Database DSN")
	fs.StringVar(&o.ArtefactDir, "artefact-dir", "", "Directory for the bytes of job artefacts")
	fs.StringVar(&o.SigningKey, "signing-key", "", "Signing key for access tokens")
	fs.StringVar(&o.AgentToken, "agent-token", "", "Shared token agents enrol with")
	fs.BoolVar(&o.AllowRegistration, "allow-registration", false, "Keep registration open after the first user")
}

// Server provides a cli.Command that runs the atkins CI/CD server.
func Server() *cli.Command {
	opts := &ServerOptions{}

	return &cli.Command{
		Name:  "server",
		Title: "Run the atkins CI/CD server",
		Bind:  opts.Bind,
		Run: func(ctx context.Context, _ []string) error {
			return runServer(ctx, opts)
		},
	}
}

func runServer(ctx context.Context, opts *ServerOptions) error {
	cwd, _ := os.Getwd()

	settings, err := loadRuntimeConfig(cwd)
	if err != nil {
		return err
	}

	config := settings.Server
	override(&config.Addr, opts.Addr)
	override(&config.Database, opts.Database)
	override(&config.ArtefactDir, opts.ArtefactDir)
	override(&config.SigningKey, opts.SigningKey)
	override(&config.AgentToken, opts.AgentToken)
	if opts.AllowRegistration {
		config.AllowRegistration = true
	}

	if config.SigningKey == "" {
		return fmt.Errorf("server.signing_key is required to sign access tokens; set it with `atkins --config`, --signing-key, or %s",
			server.EnvSigningKey)
	}

	// The platform reads PLATFORM_DB_* at package init, before the
	// configuration is resolved. Publish the DSN we settled on and
	// re-register so the config value reaches the connection provider.
	if err := os.Setenv("PLATFORM_DB_DEFAULT", config.Database); err != nil {
		return err
	}
	platform.SetupConnections(os.Environ())

	svc := platform.New(&platform.Options{ServerAddr: config.Addr})
	svc.Register(server.NewModule(server.FromConfig(config)))

	if err := svc.Start(ctx); err != nil {
		return err
	}

	svc.Wait()
	return nil
}
