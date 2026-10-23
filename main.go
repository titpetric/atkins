package main

import (
	"fmt"
	"io"
	"os"

	// The storage package leaves sql driver registration to its caller,
	// so the binary picks the one it needs. modernc's sqlite is a pure
	// Go translation and keeps CGO_ENABLED=0, which the release build in
	// atkins.yml relies on.
	_ "modernc.org/sqlite"

	"github.com/titpetric/cli"
)

func main() {
	if err := start(os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// start writes the help document for --help and runs the command otherwise.
func start(stdout io.Writer, args []string) error {
	// The cli package prints a command list and nothing else, and its
	// help hook is per command. The long document covers the flags, the
	// skills and the schema at once, so main answers --help here before
	// selecting a command.
	if HelpRequested(HelpFlags(), args) {
		return WriteHelp(stdout)
	}

	app := cli.NewApp("atkins")
	app.AddCommand("run", "Run pipeline", Pipeline)
	app.DefaultCommand = "run"
	return app.Run()
}
