// Package helpdoc renders `atkins --help`.
//
// The help is a document rather than a flag dump: the usage line, the
// commands, every flag with its default, the skills installed on this
// machine with the jobs each contributes, and a reference for the
// atkins.yml schema. Someone reading it, or an agent, should be able to
// run atkins and write a pipeline without opening a second page.
//
// The document is markdown throughout, with padded table cells so it
// reads as columns in a terminal too. Colour is added to the headings
// only when the caller says the writer is a terminal, so
// `atkins --help > help.md` is a document and not a screenshot of one.
package helpdoc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// Write renders the whole help document.
//
// The sections are ordered so a reader can stop early: what to type,
// then the flags, then what the installed skills offer, then the schema
// for writing a pipeline of your own.
func Write(w io.Writer, opts *Options) error {
	name := opts.Name
	if name == "" {
		name = "atkins"
	}

	d := &document{w: w, color: opts.Color}

	d.heading(1, name)
	if opts.Version != "" {
		d.text(opts.Version)
	}
	d.raw(fmt.Sprintf("```\nUsage: %s [flags] [job...]\n       %s <command> [flags]\n```", name, name))
	d.gap()
	d.text(fmt.Sprintf(
		"A job is a name from atkins.yml or from a skill, `%s go:fmt`. Several may be named at once. With none, %s runs the job called `default`. `%s -l` lists them.",
		name, name, name))

	d.commands(opts.Commands)
	d.flags(opts.Commands)
	d.skills(opts)

	d.heading(2, "atkins.yml")
	d.raw(Schema())

	return nil
}

// document writes the help sections to one writer.
type document struct {
	w       io.Writer
	color   bool
	started bool
	blank   bool
}

// gap writes the blank line that separates two blocks, and writes
// nothing when the last block already ended with one. Every block
// spacing goes through it, so a section that ends in a table and one
// that ends in a paragraph are followed by the same single blank line.
func (d *document) gap() {
	if d.started && !d.blank {
		fmt.Fprintln(d.w)
	}
	d.blank = true
}

// heading writes a markdown heading, coloured for a terminal. Every
// heading but the first opens with a blank line, which is what separates
// the sections; the document itself starts at its first byte.
func (d *document) heading(level int, text string) {
	line := strings.Repeat("#", level) + " " + text
	if d.color {
		switch level {
		case 1:
			line = colors.BrightCyan(line)
		default:
			line = colors.BrightWhite(line)
		}
	}

	d.gap()
	d.started = true

	fmt.Fprintf(d.w, "%s\n\n", line)
	d.blank = true
}

// text writes one paragraph.
func (d *document) text(text string) {
	fmt.Fprintf(d.w, "%s\n\n", text)
	d.started = true
	d.blank = true
}

// raw writes a block that is already markdown.
func (d *document) raw(text string) {
	fmt.Fprintf(d.w, "%s\n", strings.TrimSpace(text))
	d.started = true
	d.blank = false
}

// commands writes the subcommand table.
func (d *document) commands(commands []Command) {
	if len(commands) == 0 {
		return
	}

	d.heading(2, "Commands")

	t := newTable(d.w, "Command", "What it does")
	for _, command := range commands {
		title := command.Title
		if command.Default {
			title += " (the default, so the word is optional)"
		}
		t.row("`"+command.Name+"`", title)
	}
	t.flush()
	d.blank = false
}

// flags writes one flag table per command, the default command's under
// the heading a reader looks for it by.
func (d *document) flags(commands []Command) {
	for _, command := range commands {
		if command.Flags == nil || !command.Flags.HasFlags() {
			continue
		}

		if command.Default {
			d.heading(2, "Flags")
		} else {
			d.heading(2, "Flags for "+command.Name)
		}

		t := newTable(d.w, "Flag", "Default", "What it does")
		command.Flags.VisitAll(func(f *pflag.Flag) {
			name := "--" + f.Name
			if f.Shorthand != "" {
				name = "-" + f.Shorthand + ", " + name
			}

			value := f.DefValue
			switch value {
			case "", "false", "0", "0s", "[]":
				value = ""
			default:
				value = "`" + value + "`"
			}

			t.row("`"+name+"`", value, f.Usage)
		})
		t.flush()
		d.blank = false
	}
}

// skills writes where skills live and what each installed one offers.
func (d *document) skills(opts *Options) {
	d.heading(2, "Skills")

	d.text("A skill is a pipeline file whose name is a job namespace: `go.yml` contributes `go:build`, `go:test` and the rest, addressed as `atkins go:test`. A skill with a `default` job answers to its bare name, `atkins go`, and a job may declare `aliases` for shorter ones.")

	if len(opts.SkillDirs) > 0 {
		lines := make([]string, len(opts.SkillDirs))
		for i, dir := range opts.SkillDirs {
			lines[i] = "- `" + shorten(dir) + "`"
		}
		d.text("Skills are read from, in precedence order:\n\n" + strings.Join(lines, "\n"))
	}

	d.text("The first directory wins: a repository's own `go.yml` shadows the one in `$HOME`. `atkins --vendor` reports which skills a repository uses and `--write` copies them into `.atkins/skills` next to `.git`, so a clone or a CI agent runs the same jobs. `--jail` ignores `$HOME` and runs on the repository's skills alone.")

	d.text("A skill may carry a markdown file of the same name beside it, `go.md` beside `go.yml`, which is printed below as its usage guide. It travels with the skill when vendored.")

	skills := dedupe(opts.Skills)
	if len(skills) == 0 {
		d.text("No skills are installed.")
		return
	}

	t := newTable(d.w, "Skill", "Purpose", "Read from")
	for _, skill := range skills {
		t.row("`"+skill.ID()+"`", firstSentence(purpose(skill)), "`"+shorten(filepath.Dir(skill.Path))+"`")
	}
	t.flush()
	d.blank = false

	for _, skill := range skills {
		d.skill(skill)
	}
}

// skill writes one skill: its purpose, its guide when it carries one,
// and the jobs decoded from its YAML.
func (d *document) skill(skill *runner.Skill) {
	d.heading(3, skill.ID())

	if text := purpose(skill); text != "" {
		d.text(text)
	}
	if !skill.Active && skill.Pipeline.When != nil {
		d.text("Not active here: nothing in this directory or above it matches `when: files: [" +
			strings.Join(skill.Pipeline.When.Files, ", ") + "]`.")
	}
	if skill.Doc != "" {
		d.raw(skill.Doc)
		d.gap()
	}

	jobs := skill.Pipeline.GetJobs()
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}

	// The default job runs when the skill is named on its own, so it
	// heads the list; the rest are alphabetical.
	sort.Slice(names, func(i, j int) bool {
		if (names[i] == "default") != (names[j] == "default") {
			return names[i] == "default"
		}
		return names[i] < names[j]
	})

	t := newTable(d.w, "Job", "What it does")
	for _, name := range names {
		job := jobs[name]

		target := skill.ID() + ":" + name
		if name == "default" {
			target = skill.ID()
		}

		t.row("`atkins "+target+"`", describe(job))
	}
	t.flush()
	d.blank = false
}

// describe returns a job's one-line description, falling back to its
// dependencies when it has none of its own.
func describe(job *model.Job) string {
	if job.Desc != "" {
		return job.Desc
	}
	if len(job.DependsOn) > 0 {
		return "runs " + strings.Join(job.DependsOn, ", ")
	}
	return ""
}

// purpose returns what a skill says it is for: its help: line, or the
// pipeline name when it carries none.
func purpose(skill *runner.Skill) string {
	if skill.Pipeline.Help != "" {
		return skill.Pipeline.Help
	}
	return skill.Pipeline.Name
}

// firstSentence returns the opening sentence of a help line, which is
// what the overview table carries; the skill's own section prints the
// whole of it.
func firstSentence(text string) string {
	for i, r := range text {
		if r != '.' || i+1 >= len(text) {
			continue
		}
		if text[i+1] == ' ' || text[i+1] == '\n' {
			return text[:i+1]
		}
	}
	return text
}

// dedupe keeps the first skill seen per ID, which is the one a run would
// use, and orders the result by ID.
func dedupe(skills []*runner.Skill) []*runner.Skill {
	seen := make(map[string]bool, len(skills))

	var result []*runner.Skill
	for _, skill := range skills {
		if skill == nil || skill.Pipeline == nil || seen[skill.ID()] {
			continue
		}
		seen[skill.ID()] = true
		result = append(result, skill)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})

	return result
}

// shorten writes a path the way a reader would type it: relative to the
// working directory when it is inside it, under ~ when it is inside the
// home directory, absolute otherwise.
func shorten(path string) string {
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}

	return path
}
