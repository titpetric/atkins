package main

import (
	"fmt"
	"os"

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

	app := cli.NewApp("atkins")
	app.AddCommand("run", "Run pipeline", Pipeline)
	app.AddCommand("server", "Run the CI/CD server", Server)
	app.AddCommand("worker", "Run the CI/CD agent", Worker)
	app.DefaultCommand = "run"
	return app.Run()
}
