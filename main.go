package main

import (
	"fmt"
	"os"

	"github.com/titpetric/cli"
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

	app.DefaultCommand = "run"

	return app.Run()
}
