package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/pipeline"
)

// listing is what `atkins --list --json` writes for a project with a
// pipeline of its own and one skill alongside it.
const listing = `[
  {
    "desc": "Atkins project tests, build",
    "cmds": [
      {"id": "default", "desc": "Run everything needed", "cmd": "atkins default"},
      {"id": "shell", "desc": "A shell on the agent", "cmd": "atkins shell", "interactive": true},
      {"id": "test:cover", "desc": "Coverage", "cmd": "atkins test:cover", "depends_on": ["test:simple"]},
      {"id": "test:simple", "desc": "Tests", "cmd": "atkins test:simple"}
    ]
  },
  {
    "desc": "Docker compose lifecycle",
    "cmds": [
      {"id": "compose:up", "cmd": "atkins compose:up"}
    ]
  }
]`

func TestParseBuildsATreeFromColonNames(t *testing.T) {
	tree, err := pipeline.Parse([]byte(listing))
	require.NoError(t, err)
	require.Len(t, tree.Sections, 2)

	project := tree.Sections[0]
	assert.Equal(t, "Atkins project tests, build", project.Name)

	// `default`, `shell` and the `test` grouping. The order atkins listed
	// them in is kept, which is what puts `default` first.
	require.Len(t, project.Nodes, 3)
	assert.Equal(t, "default", project.Nodes[0].Label)
	assert.Equal(t, "shell", project.Nodes[1].Label)

	// `test` is a grouping and nothing else: the pipeline declares
	// `test:cover` and `test:simple` but no `test` of its own, so
	// choosing it would dispatch a job that does not exist.
	group := project.Nodes[2]
	assert.Equal(t, "test", group.Label)
	assert.False(t, group.Runnable())
	require.Len(t, group.Children, 2)
	assert.Equal(t, "cover", group.Children[0].Label)
	assert.True(t, group.Children[0].Runnable())
	assert.Equal(t, "test:cover", group.Children[0].Command.ID)
}

// A node is a grouping and a choice at once when the pipeline declares
// both. Rendering it as only one of the two would either hide the job or
// hide its children.
func TestParseKeepsAJobThatIsAlsoAGroup(t *testing.T) {
	tree, err := pipeline.Parse([]byte(`[{"desc":"p","cmds":[
		{"id":"test","cmd":"atkins test"},
		{"id":"test:cover","cmd":"atkins test:cover"}
	]}]`))
	require.NoError(t, err)

	node := tree.Sections[0].Nodes[0]
	assert.Equal(t, "test", node.Label)
	assert.True(t, node.Runnable())
	require.Len(t, node.Children, 1)
	assert.Equal(t, "cover", node.Children[0].Label)
}

// The interactive flag is the whole reason the listing carries more than
// names: it decides whether the terminal a browser opens types back.
func TestParseCarriesTheInteractiveFlag(t *testing.T) {
	tree, err := pipeline.Parse([]byte(listing))
	require.NoError(t, err)

	shell, found := tree.Lookup("shell")
	require.True(t, found)
	assert.True(t, shell.Interactive)

	tests, found := tree.Lookup("test:simple")
	require.True(t, found)
	assert.False(t, tests.Interactive)
}

// Dispatch looks a job up here rather than pasting the form's value into
// a command, so a name the project does not declare cannot be run.
func TestLookupRefusesAJobThePipelineDoesNotDeclare(t *testing.T) {
	tree, err := pipeline.Parse([]byte(listing))
	require.NoError(t, err)

	_, found := tree.Lookup("rm -rf /")
	assert.False(t, found)

	_, found = tree.Lookup("compose:up")
	assert.True(t, found)
}

// A job in the project shadows a skill's job of the same name, because
// that is what running it would do.
func TestLookupPrefersTheFirstSection(t *testing.T) {
	tree, err := pipeline.Parse([]byte(`[
		{"desc":"project","cmds":[{"id":"fmt","desc":"the project's","cmd":"atkins fmt"}]},
		{"desc":"skill","cmds":[{"id":"fmt","desc":"the skill's","cmd":"atkins fmt"}]}
	]`))
	require.NoError(t, err)

	command, found := tree.Lookup("fmt")
	require.True(t, found)
	assert.Equal(t, "the project's", command.Desc)
}

// A listing that named nothing means the agent ran somewhere without a
// pipeline. An empty tree would say "this project has no jobs", which
// sends whoever reads it looking in the wrong place.
func TestParseRefusesAnEmptyListing(t *testing.T) {
	for _, listing := range []string{``, `[]`, `[{"desc":"p","cmds":[]}]`, `not json`} {
		_, err := pipeline.Parse([]byte(listing))
		assert.Error(t, err, listing)
	}
}

// The artefact is whatever the job's shell redirected into it, so a
// warning printed by something earlier in the agent's PATH must not cost
// the whole listing.
func TestParseSkipsWhatWasPrintedBeforeTheJSON(t *testing.T) {
	tree, err := pipeline.Parse([]byte("warning: something\n" + listing))
	require.NoError(t, err)

	_, found := tree.Lookup("default")
	assert.True(t, found)
}

func TestCommandsIsSortedAndComplete(t *testing.T) {
	tree, err := pipeline.Parse([]byte(listing))
	require.NoError(t, err)

	ids := []string{}
	for _, command := range tree.Commands() {
		ids = append(ids, command.ID)
	}

	assert.Equal(t, []string{"compose:up", "default", "shell", "test:cover", "test:simple"}, ids)
}

// A nil tree is what a page holds before a project's pipeline has been
// read, and neither method may panic on one.
func TestNilTreeAnswersEmpty(t *testing.T) {
	var tree *pipeline.Tree

	_, found := tree.Lookup("default")
	assert.False(t, found)
	assert.Empty(t, tree.Commands())
}
