package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/runner"
)

// runVendor handles `atkins --vendor`, which copies the skills this
// repository uses out of $HOME/.atkins/skills and into .atkins/skills
// next to .git, so a clone or a CI agent runs the same jobs as the
// machine the pipeline was written on.
func runVendor(dir string, opts *Options) error {
	if opts.Jail {
		return fmt.Errorf("%s --vendor reads $HOME/.atkins/skills, which --jail excludes", colors.BrightRed("ERROR:"))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("%s failed to resolve home directory: %v", colors.BrightRed("ERROR:"), err)
	}

	root, err := runner.FindRepositoryRoot(dir)
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}

	vendorer := runner.NewVendorer(filepath.Join(home, ".atkins", "skills"), root)

	result, err := vendorer.Run()
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}

	fmt.Printf("Found %d local skills.\n", len(result.Found))
	fmt.Printf("Found usage for %d skills.\n", len(result.Used))

	if opts.Debug {
		for _, skill := range result.Used {
			fmt.Printf("  %s: %s\n", colors.BrightOrange(skill.ID), colors.Dim(skill.Reason))
		}
	}

	if len(result.Installed) > 0 {
		fmt.Printf("Installed: %s.\n", strings.Join(result.Installed, ", "))
	}

	// A project skill wins over a global one at load time, so a local
	// copy that has drifted is a deliberate override, not a stale file.
	if len(result.Skipped) > 0 {
		target := result.TargetDir
		if rel, err := filepath.Rel(root, result.TargetDir); err == nil {
			target = rel
		}
		fmt.Printf("Skipped: %s (%s carries its own copy).\n", strings.Join(result.Skipped, ", "), target)
	}

	return nil
}
