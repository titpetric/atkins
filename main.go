package main

import (
	"fmt"
	"os"

	// The platform leaves sql driver registration to its callers, so the
	// binary picks the one it needs. modernc's
	// sqlite is a pure Go translation and keeps CGO_ENABLED=0, which the
	// release build in atkins.yml relies on.
	_ "modernc.org/sqlite"

	"github.com/titpetric/cli"

	"github.com/titpetric/atkins/client"
)

func main() {
	if err := start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func start() error {
	client.UserAgent = "atkins/" + Version

	// The cli package prints a command list and nothing else, and its
	// help hook is per command. The long document covers the flags, the
	// skills and the schema at once, so main answers --help here before
	// selecting a command.
	if HelpRequested(HelpFlags(), os.Args[1:]) {
		return WriteHelp(os.Stdout)
	}

	app := cli.NewApp("atkins")
	app.AddCommand("run", "Run pipeline", Pipeline)
	app.AddCommand("server", "Run the CI/CD server", Server)
	app.AddCommand("worker", "Run the CI/CD agent", Worker)
	app.DefaultCommand = "run"
	return app.Run()
}
