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
//
// It reports what it would do and writes nothing until --write is
// given, so the selection can be read before the tree changes.
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

	result, err := vendorer.Plan()
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}

	if opts.Write {
		if err := vendorer.Install(result); err != nil {
			return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
		}
	}

	fmt.Printf("Found %d local skills.\n", len(result.Found))
	fmt.Printf("Found usage for %d skills.\n", len(result.Used))

	width := 0
	for _, skill := range result.Used {
		if len(skill.ID) > width {
			width = len(skill.ID)
		}
	}
	for _, skill := range result.Used {
		printVendorSkill(skill, width, opts.Debug)
	}

	pending := result.Pending()
	switch {
	case len(result.Used) == 0:
		fmt.Println("Nothing to install.")
	case len(pending) == 0:
		fmt.Println("Every skill is up to date.")
	case opts.Write:
		fmt.Printf("Installed: %s.\n", strings.Join(result.Installed, ", "))
	default:
		fmt.Printf("Would install: %s.\n", strings.Join(pending, ", "))
		fmt.Printf("Run %s to write them.\n", colors.BrightOrange("atkins --vendor --write"))
	}

	return nil
}

// printVendorSkill reports one selected skill: a checkmark when the
// vendored copy already matches, and the lines a write would add and
// remove when it doesn't. Debug adds the reason the skill was selected
// and the diff the counts come from.
func printVendorSkill(skill *runner.VendorSkill, width int, debug bool) {
	reason := ""
	if debug {
		reason = " " + colors.Dim("("+skill.Reason+")")
	}

	name := fmt.Sprintf("%-*s", width, skill.ID)

	if skill.Status == runner.VendorCurrent {
		fmt.Printf("  %s %s %s%s\n", colors.BrightGreen("✓"), name, colors.Dim("(up to date)"), reason)
		return
	}

	marker := colors.BrightYellow("~")
	if skill.Status == runner.VendorNew {
		marker = colors.BrightGreen("+")
	}

	stats := fmt.Sprintf("%s %s",
		colors.BrightGreen(fmt.Sprintf("+%d", skill.Added)),
		colors.BrightRed(fmt.Sprintf("-%d", skill.Removed)),
	)

	fmt.Printf("  %s %s %s %s%s\n",
		marker, name, stats, colors.Dim("("+string(skill.Status)+")"), reason)

	if debug && skill.Diff != "" {
		for _, line := range strings.Split(strings.TrimRight(skill.Diff, "\n"), "\n") {
			fmt.Printf("    %s\n", colorDiffLine(line))
		}
	}
}

// colorDiffLine paints a unified diff line by what it does.
func colorDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		return colors.Dim(line)
	case strings.HasPrefix(line, "@@"):
		return colors.BrightCyan(line)
	case strings.HasPrefix(line, "+"):
		return colors.BrightGreen(line)
	case strings.HasPrefix(line, "-"):
		return colors.BrightRed(line)
	}
	return colors.Dim(line)
}
