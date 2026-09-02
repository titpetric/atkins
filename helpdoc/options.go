package helpdoc

import "github.com/titpetric/atkins/runner"

// Options is what Write renders.
type Options struct {
	// Name is the executable name in the usage line.
	Name string

	// Version is printed under the usage line, empty to leave it out.
	Version string

	// Commands are the subcommands, the default one first.
	Commands []Command

	// Skills are the skill files found on disk, in precedence order.
	// Write deduplicates them by ID and reports the ones the current
	// directory does not activate.
	Skills []*runner.Skill

	// SkillDirs are the directories the skills were read from, in
	// precedence order, printed so a reader knows where to add one.
	SkillDirs []string

	// Color turns on ANSI colour for headings. Callers set it from a
	// terminal check on the writer, so a redirected help document is
	// plain markdown.
	Color bool
}
