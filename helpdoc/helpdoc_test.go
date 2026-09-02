package helpdoc_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/helpdoc"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// testOptions builds a document with one command and two skills, one of
// them inactive and one carrying a markdown companion.
func testOptions() *helpdoc.Options {
	flags := pflag.NewFlagSet("run", pflag.ContinueOnError)
	var list bool
	var file string
	flags.BoolVarP(&list, "list", "l", false, "List pipeline jobs and dependencies")
	flags.StringVarP(&file, "file", "f", "atkins.yml", "Path to pipeline file")

	return &helpdoc.Options{
		Name:      "atkins",
		Version:   "Version dev.",
		SkillDirs: []string{".atkins/skills", "/home/user/.atkins/skills"},
		Commands: []helpdoc.Command{
			{Name: "run", Title: "Run jobs from a pipeline file", Default: true, Flags: flags},
			{Name: "server", Title: "Run the CI/CD server"},
		},
		Skills: []*runner.Skill{
			{
				Path:   "/home/user/.atkins/skills/go.yml",
				Doc:    "`atkins go:fmt` formats the sources.",
				Active: true,
				Pipeline: &model.Pipeline{
					ID:   "go",
					Name: "Go build and test",
					Help: "Builds, formats, lints and tests a Go module.",
					Jobs: map[string]*model.Job{
						"fmt":     {Desc: "Format the code"},
						"default": {DependsOn: model.Dependencies{"fmt", "test"}},
					},
				},
			},
			{
				Path:   "/home/user/.atkins/skills/tailwind.yml",
				Active: false,
				Pipeline: &model.Pipeline{
					ID:    "tailwind",
					Name:  "Tailwind CSS generation",
					When:  &model.PipelineWhen{Files: []string{"tailwind.config.js"}},
					Jobs:  map[string]*model.Job{"generate": {Desc: "Generate Tailwind CSS"}},
					Tasks: nil,
				},
			},
		},
	}
}

// TestWrite renders the whole document and checks every section is in
// it, in the order a reader can stop reading at.
func TestWrite(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, helpdoc.Write(&out, testOptions()))

	doc := out.String()

	sections := []string{
		"# atkins",
		"Usage: atkins [flags] [job...]",
		"## Commands",
		"## Flags",
		"## Skills",
		"### go",
		"### tailwind",
		"## atkins.yml",
		"### One idea, several spellings",
	}

	offset := 0
	for _, section := range sections {
		index := strings.Index(doc[offset:], section)
		require.GreaterOrEqual(t, index, 0, "%q missing or out of order", section)
		offset += index
	}
}

// TestWriteRendersSkills checks a skill contributes its purpose, its
// markdown companion and its jobs, and that an inactive one says so.
func TestWriteRendersSkills(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, helpdoc.Write(&out, testOptions()))

	doc := out.String()

	assert.Contains(t, doc, "Builds, formats, lints and tests a Go module.")
	assert.Contains(t, doc, "`atkins go:fmt` formats the sources.")
	assert.Contains(t, doc, "`atkins go:fmt`")
	assert.Contains(t, doc, "Format the code")

	// A default job is addressed by the bare skill name.
	assert.Contains(t, doc, "`atkins go`")
	assert.Contains(t, doc, "runs fmt, test")

	// A skill without help: falls back to its pipeline name.
	assert.Contains(t, doc, "Tailwind CSS generation")
	assert.Contains(t, doc, "Not active here")
	assert.Contains(t, doc, "tailwind.config.js")
}

// TestWriteIsPlainWithoutColor checks a document written for a file
// carries no escape sequences, and that Color adds them.
func TestWriteIsPlainWithoutColor(t *testing.T) {
	opts := testOptions()

	var plain bytes.Buffer
	require.NoError(t, helpdoc.Write(&plain, opts))
	assert.NotContains(t, plain.String(), "\x1b[")

	opts.Color = true

	var colored bytes.Buffer
	require.NoError(t, helpdoc.Write(&colored, opts))
	assert.Contains(t, colored.String(), "\x1b[")
}

// TestWriteFlagTable checks a flag reaches the table with its shorthand
// and its default, and that an empty default stays empty.
func TestWriteFlagTable(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, helpdoc.Write(&out, testOptions()))

	doc := out.String()

	assert.Contains(t, doc, "`-l, --list`")
	assert.Contains(t, doc, "`-f, --file`")
	assert.Contains(t, doc, "`atkins.yml`")
	assert.Contains(t, doc, "List pipeline jobs and dependencies")
}

// TestWriteWithoutSkills checks the section says so rather than printing
// an empty table.
func TestWriteWithoutSkills(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, helpdoc.Write(&out, &helpdoc.Options{}))

	doc := out.String()

	assert.Contains(t, doc, "# atkins")
	assert.Contains(t, doc, "No skills are installed.")
	assert.Contains(t, doc, "## atkins.yml")
}
