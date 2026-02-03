package main

import (
	"fmt"
	"os"

	"github.com/titpetric/cli"

	"github.com/titpetric/atkins/version"
)

func main() {
	if err := start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func start() error {
	app := cli.NewApp("atkins")
	app.AddCommand("run", "Run pipeline", Pipeline)
	app.AddCommand("version", version.Name, func() *cli.Command {
		return version.NewCommand(version.Info{
			Version:    Version,
			Commit:     Commit,
			CommitTime: CommitTime,
			Branch:     Branch,
		})
	})

	app.DefaultCommand = "run"

	return app.Run()
}
