// Package pipeline turns a project's job listing into the tree the
// browser picks a run from.
//
// The listing is `atkins --list --json`, produced by an agent in a
// checkout of the project and uploaded as an artefact. Reading it there
// rather than parsing atkins.yml on the server is what keeps the server
// out of the business of cloning: the agent already has the repository
// cache, the deploy keys and the allowlist, and a listing produced
// anywhere else would be a listing of a different checkout.
//
// What arrives is flat — one entry per invocable job, named `test:cover`
// and `release:publish`. Jobs are named that way because they are a
// hierarchy, so this package puts the hierarchy back: `test:cover` is
// `cover` under `test`, and a menu can be a tree rather than a hundred
// lines of one list.
package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/titpetric/atkins/runner"
)

// Command is one invocable job in a project's pipeline.
type Command struct {
	// ID is the job name atkins invokes, `test:cover` and such.
	ID string `json:"id"`

	// Desc is the job's `desc:`, empty for a job that has none.
	Desc string `json:"desc,omitempty"`

	// Command is the full invocation, `atkins test:cover`.
	Command string `json:"command"`

	// Interactive reports that the job binds stdin. It decides whether
	// the terminal a browser opens on the run types back.
	Interactive bool `json:"interactive,omitempty"`

	// DependsOn are the jobs that run first, shown so a target says what
	// it drags in with it.
	DependsOn []string `json:"depends_on,omitempty"`

	// Section is the pipeline the job came from: the project's own file,
	// or one of the skills the agent resolved alongside it.
	Section string `json:"section"`
}

// Node is one entry in the tree a menu renders.
//
// A node is a job, a grouping, or both: `test` is a grouping when the
// pipeline only defines `test:cover` and `test:simple`, and is both when
// it also defines a `test` of its own.
type Node struct {
	// Label is this level's name segment, `cover` in `test:cover`.
	Label string `json:"label"`

	// Command is the job this node runs, nil for a node that only groups
	// the ones below it.
	Command *Command `json:"command,omitempty"`

	Children []*Node `json:"children,omitempty"`
}

// Runnable reports whether picking this node dispatches anything.
func (n *Node) Runnable() bool {
	return n != nil && n.Command != nil
}

// Section is one pipeline's worth of jobs, as a tree.
type Section struct {
	// Name is the pipeline's `name:`, or "Aliases" for the section
	// atkins synthesises from job aliases.
	Name string `json:"name"`

	Nodes []*Node `json:"nodes"`
}

// Tree is a whole project's pipeline, ready to render.
type Tree struct {
	Sections []Section `json:"sections"`

	// commands indexes every job by ID, so dispatching a chosen ID can
	// check it against the listing rather than trusting the form.
	commands map[string]Command
}

// Parse reads the JSON `atkins --list --json` wrote.
//
// A listing with no jobs at all is an error rather than an empty tree:
// it means the agent ran in a directory with no pipeline in it, and a
// page saying "this project has no jobs" would send whoever reads it
// looking in the wrong place.
func Parse(data []byte) (*Tree, error) {
	var sections []runner.OutputSection
	if err := json.Unmarshal(trimBanner(data), &sections); err != nil {
		return nil, fmt.Errorf("read the pipeline listing: %w", err)
	}

	tree := &Tree{commands: map[string]Command{}}

	for _, section := range sections {
		name := strings.TrimSpace(section.Desc)
		if name == "" {
			name = "Pipeline"
		}

		commands := make([]Command, 0, len(section.Cmds))
		for _, item := range section.Cmds {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}

			command := Command{
				ID:          id,
				Desc:        item.Desc,
				Command:     item.Cmd,
				Interactive: item.Interactive,
				DependsOn:   item.DependsOn,
				Section:     name,
			}
			commands = append(commands, command)

			// First writer wins. A job defined in the project shadows a
			// skill's job of the same name, and the project's section is
			// the one atkins lists first.
			if _, seen := tree.commands[id]; !seen {
				tree.commands[id] = command
			}
		}

		if len(commands) == 0 {
			continue
		}

		tree.Sections = append(tree.Sections, Section{Name: name, Nodes: build(commands)})
	}

	if len(tree.commands) == 0 {
		return nil, fmt.Errorf("the pipeline listing names no jobs")
	}

	return tree, nil
}

// Lookup returns the job with an ID, and whether the listing has one.
//
// Dispatch goes through here rather than taking the form's word for it:
// the value arrives from a browser, and a project's runnable jobs are
// exactly the ones its own pipeline declares.
func (t *Tree) Lookup(id string) (Command, bool) {
	if t == nil {
		return Command{}, false
	}
	command, ok := t.commands[strings.TrimSpace(id)]
	return command, ok
}

// Commands returns every job in the listing, sorted by ID. It is what a
// caller wanting a flat list — a datalist, an API response — reads.
func (t *Tree) Commands() []Command {
	if t == nil {
		return nil
	}

	commands := make([]Command, 0, len(t.commands))
	for _, command := range t.commands {
		commands = append(commands, command)
	}
	sort.Slice(commands, func(i, j int) bool { return commands[i].ID < commands[j].ID })

	return commands
}

// build turns a flat list of `a:b:c` names into the tree they describe.
//
// The order jobs arrive in is the order atkins lists them, which puts
// `default` first and is worth keeping: it is the job somebody most
// likely wants. Children are therefore appended as they are first seen
// rather than sorted.
func build(commands []Command) []*Node {
	var roots []*Node
	index := map[string]*Node{}

	for _, command := range commands {
		segments := strings.Split(command.ID, ":")

		var (
			parent *Node
			prefix string
		)

		for depth, segment := range segments {
			if depth > 0 {
				prefix += ":"
			}
			prefix += segment

			node, found := index[prefix]
			if !found {
				node = &Node{Label: segment}
				index[prefix] = node

				if parent == nil {
					roots = append(roots, node)
				} else {
					parent.Children = append(parent.Children, node)
				}
			}

			// Only the last segment names the job itself; the ones before
			// it are the groupings that job hangs under.
			if depth == len(segments)-1 {
				assigned := command
				node.Command = &assigned
			}

			parent = node
		}
	}

	return roots
}

// trimBanner drops anything before the JSON document.
//
// `atkins --list --json` prints the listing and nothing else, but the
// artefact is whatever the agent's shell redirected into the file, and a
// warning from a tool earlier in the job's PATH would otherwise make the
// whole listing unreadable. Finding the first `[` costs nothing and
// turns a class of failure into a working page.
func trimBanner(data []byte) []byte {
	if start := bytes.IndexByte(data, '['); start > 0 {
		return data[start:]
	}
	return data
}
