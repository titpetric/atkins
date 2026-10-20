package main

import (
	"fmt"
	"os"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/config"
)

// runtimeConfig is the effective configuration for this invocation:
// the embedded document from `config.RuntimeDefaultConfig`, overlaid with
// the user and project documents, then with the environment.
//
// atkins loads it once and shares it, so every command sees the same values
// and a configuration error surfaces once and never per subsystem.
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

// runConfig runs the configuration menu.
//
// It edits the project document (.atkins/config.yml, created from the
// embedded default when absent) and leaves the merged view alone, so the
// saved file holds only what this project chose.
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
