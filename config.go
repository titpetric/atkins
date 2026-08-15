package main

import (
	"fmt"
	"os"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/config"
)

// runtimeConfig is the effective configuration for this invocation:
// the embedded document from config.RuntimeDefaultConfig, overlaid with
// the user and project documents, then with the environment.
//
// It is loaded once and shared, so every command sees the same values
// and a config error is reported once rather than per subsystem.
var runtimeConfig *config.Config

// loadRuntimeConfig resolves the configuration for a working directory.
//
// A broken document is fatal: the alternative is a run that silently
// uses defaults the user thought they had changed.
func loadRuntimeConfig(dir string) (*config.Config, error) {
	if runtimeConfig != nil {
		return runtimeConfig, nil
	}

	loaded, err := config.Load(dir)
	if err != nil {
		return nil, err
	}

	runtimeConfig = loaded
	return runtimeConfig, nil
}

// runConfig handles `atkins --config`, the configuration menu.
//
// It edits the project document — .atkins/config.yml, created from the
// embedded default when absent — rather than the merged view, so what
// is written back is only what this project chose.
func runConfig(dir string) error {
	path := config.ProjectPath(dir)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s %s does not exist yet; starting from the built-in defaults\n",
			colors.BrightYellow("INFO:"), colors.Dim(path))
	}

	menu, err := config.NewMenu(path)
	if err != nil {
		return err
	}

	return menu.Run()
}
