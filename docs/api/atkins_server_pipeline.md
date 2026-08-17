# Package ./server/pipeline

```go
import (
	"github.com/titpetric/atkins/server/pipeline"
}
```

Package pipeline turns a project's job listing into the tree the
browser picks a run from.

The listing is `atkins --list --json`, produced by an agent in a
checkout of the project and uploaded as an artefact. Reading it there
rather than parsing atkins.yml on the server is what keeps the server
out of the business of cloning: the agent already has the repository
cache, the deploy keys and the allowlist, and a listing produced
anywhere else would be a listing of a different checkout.

What arrives is flat — one entry per invocable job, named `test:cover`
and `release:publish`. Jobs are named that way because they are a
hierarchy, so this package puts the hierarchy back: `test:cover` is
`cover` under `test`, and a menu can be a tree rather than a hundred
lines of one list.

## Types

```go
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
```

```go
// Node is one entry in the tree a menu renders.
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
```

```go
// Section is one pipeline's worth of jobs, as a tree.
type Section struct {
	// Name is the pipeline's `name:`, or "Aliases" for the section
	// atkins synthesises from job aliases.
	Name string `json:"name"`

	Nodes []*Node `json:"nodes"`
}
```

```go
// Tree is a whole project's pipeline, ready to render.
type Tree struct {
	Sections []Section `json:"sections"`

	// commands indexes every job by ID, so dispatching a chosen ID can
	// check it against the listing rather than trusting the form.
	commands map[string]Command
}
```

## Function symbols

- `func Parse (data []byte) (*Tree, error)`
- `func (*Node) Runnable () bool`
- `func (*Tree) Commands () []Command`
- `func (*Tree) Lookup (id string) (Command, bool)`

### Parse

Parse reads the JSON `atkins --list --json` wrote.

A listing with no jobs at all is an error rather than an empty tree:
it means the agent ran in a directory with no pipeline in it, and a
page saying "this project has no jobs" would send whoever reads it
looking in the wrong place.

```go
func Parse(data []byte) (*Tree, error)
```

### Runnable

Runnable reports whether picking this node dispatches anything.

```go
func (*Node) Runnable() bool
```

### Commands

Commands returns every job in the listing, sorted by ID. It is what a
caller wanting a flat list — a datalist, an API response — reads.

```go
func (*Tree) Commands() []Command
```

### Lookup

Lookup returns the job with an ID, and whether the listing has one.

Dispatch goes through here rather than taking the form's word for it:
the value arrives from a browser, and a project's runnable jobs are
exactly the ones its own pipeline declares.

```go
func (*Tree) Lookup(id string) (Command, bool)
```
