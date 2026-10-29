package runner

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

func TestBuildListOutput(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID:   "",
		Name: "Main Pipeline",
		Jobs: map[string]*model.Job{
			"build":   {Desc: "Build the project"},
			"default": {Desc: "Default task"},
		},
	}

	goSkill := &model.Pipeline{
		ID:   "go",
		Name: "Go Skill",
		Jobs: map[string]*model.Job{
			"test":    {Desc: "Run Go tests"},
			"default": {Desc: "Default Go task"},
		},
	}

	pipelines := []*model.Pipeline{mainPipeline, goSkill}
	output := buildListOutput(pipelines)

	// Should have 3 sections: main, aliases, go skill
	if len(output) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(output))
	}

	// First section is the main pipeline
	if output[0].Desc != "Main Pipeline" {
		t.Errorf("expected first section to be 'Main Pipeline', got %s", output[0].Desc)
	}

	// Main pipeline should have "default" first
	if len(output[0].Cmds) < 2 {
		t.Fatal("expected at least 2 commands in main pipeline")
	}
	if output[0].Cmds[0].ID != "default" {
		t.Errorf("expected first command to be 'default', got %s", output[0].Cmds[0].ID)
	}

	// Second section is aliases
	if output[1].Desc != "Aliases" {
		t.Errorf("expected second section to be 'Aliases', got %s", output[1].Desc)
	}

	// Third section is the go skill
	if output[2].Desc != "Go Skill" {
		t.Errorf("expected third section to be 'Go Skill', got %s", output[2].Desc)
	}
}

func TestBuildListOutput_EmptyPipelines(t *testing.T) {
	output := buildListOutput(nil)
	if output != nil {
		t.Errorf("expected nil for empty pipelines, got %v", output)
	}
}

func TestBuildListOutput_NoMainPipeline(t *testing.T) {
	goSkill := &model.Pipeline{
		ID:   "go",
		Name: "Go Skill",
		Jobs: map[string]*model.Job{
			"test": {Desc: "Run Go tests"},
		},
	}

	output := buildListOutput([]*model.Pipeline{goSkill})

	// Should have 1 section (skill only, no aliases since no default)
	if len(output) != 1 {
		t.Fatalf("expected 1 section, got %d", len(output))
	}
	if output[0].Desc != "Go Skill" {
		t.Errorf("expected 'Go Skill', got %s", output[0].Desc)
	}
}

func TestBuildPipelineSection(t *testing.T) {
	p := &model.Pipeline{
		ID:   "go",
		Name: "Go Skill",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build app"},
			"test":  {Desc: "Run tests"},
		},
	}

	section := buildPipelineSection(p, "go")

	if section.Desc != "Go Skill" {
		t.Errorf("expected desc 'Go Skill', got %s", section.Desc)
	}
	if len(section.Cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(section.Cmds))
	}

	// Check that IDs have prefix
	for _, cmd := range section.Cmds {
		if !strings.HasPrefix(cmd.ID, "go:") {
			t.Errorf("expected ID to have 'go:' prefix, got %s", cmd.ID)
		}
		if !strings.HasPrefix(cmd.Cmd, "atkins go:") {
			t.Errorf("expected Cmd to have 'atkins go:' prefix, got %s", cmd.Cmd)
		}
	}
}

func TestBuildAliasesSection(t *testing.T) {
	skills := []*model.Pipeline{
		{
			ID:   "go",
			Name: "Go",
			Jobs: map[string]*model.Job{
				"default": {Desc: "Default"},
				"test":    {Desc: "Test", Aliases: []string{"t"}},
			},
		},
	}

	section := buildAliasesSection(skills)

	if section.Desc != "Aliases" {
		t.Errorf("expected desc 'Aliases', got %s", section.Desc)
	}

	// Should have 2 aliases: "go" (for default) and "t" (for test)
	if len(section.Cmds) != 2 {
		t.Fatalf("expected 2 aliases, got %d: %+v", len(section.Cmds), section.Cmds)
	}

	// Check aliases come sorted
	if section.Cmds[0].ID != "go" {
		t.Errorf("expected first alias to be 'go', got %s", section.Cmds[0].ID)
	}
	if section.Cmds[1].ID != "t" {
		t.Errorf("expected second alias to be 't', got %s", section.Cmds[1].ID)
	}
}

func TestListPipelinesJSON(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID:   "",
		Name: "Main",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build"},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w

	err = ListPipelinesJSON([]*model.Pipeline{mainPipeline})

	assert.NoError(t, w.Close())
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)
	output := buf.String()

	// Verify it's valid JSON
	var parsed []OutputSection
	assert.NoError(t, json.Unmarshal([]byte(output), &parsed), "output: %s", output)

	assert.Len(t, parsed, 1)
	assert.Equal(t, "Main", parsed[0].Desc)
}

func TestListPipelinesYAML(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID:   "",
		Name: "Main",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build"},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w

	err = ListPipelinesYAML([]*model.Pipeline{mainPipeline})

	assert.NoError(t, w.Close())
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)
	output := buf.String()

	// Verify it's valid YAML
	var parsed []OutputSection
	assert.NoError(t, yaml.Unmarshal([]byte(output), &parsed), "output: %s", output)

	assert.Len(t, parsed, 1)
	assert.Equal(t, "Main", parsed[0].Desc)
}

// TestBuildListOutput_Fields checks the section and item fields a
// caller reads: kind, path, dir, active and the when: match, the job
// name, interactive and args, and that --all brings back inactive
// skills while a job whose own when: does not match never lists.
func TestBuildListOutput_Fields(t *testing.T) {
	t.Parallel()

	mainPipeline := &model.Pipeline{
		Name: "Main",
		Path: "/ws/proj/atkins.yml",
		Jobs: map[string]*model.Job{
			"default": {Desc: "Default"},
		},
	}
	compose := &model.Pipeline{
		ID:      "compose",
		Name:    "Docker Compose",
		Path:    "/ws/.atkins/skills/compose.yml",
		Dir:     "/ws/proj",
		When:    &model.PipelineWhen{Files: []string{"compose.yml"}, Here: true},
		Matched: "/ws/proj/compose.yml",
		Jobs: map[string]*model.Job{
			"custom": {Args: true, Steps: []*model.Step{{Run: "echo first"}, {Exec: []string{"docker", "compose"}}, {Exec: []string{"true"}}}},
			"shell":  {Interactive: true},
			"build":  {Inactive: true, When: &model.PipelineWhen{Files: []string{"compose.yml"}, Contains: "build:"}},
		},
	}
	composer := &model.Pipeline{
		ID:       "composer",
		Name:     "Composer",
		Path:     "/ws/.atkins/skills/composer.yml",
		When:     &model.PipelineWhen{Files: []string{"composer.lock"}, Here: true},
		Inactive: true,
		Jobs: map[string]*model.Job{
			"update": {},
		},
	}
	pipelines := []*model.Pipeline{mainPipeline, compose, composer}

	output := buildListOutput(pipelines, OutputOptions{Dir: "/ws/proj/sub"})
	assert.Len(t, output, 2, "inactive skills stay out without --all")

	assert.Equal(t, KindPipeline, output[0].Kind)
	assert.Equal(t, "", output[0].ID)
	assert.Equal(t, "/ws/proj/atkins.yml", output[0].Path)
	assert.Equal(t, "/ws/proj/sub", output[0].Dir)
	assert.True(t, output[0].Active)

	section := output[1]
	assert.Equal(t, KindSkill, section.Kind)
	assert.Equal(t, "compose", section.ID)
	assert.Equal(t, "/ws/.atkins/skills/compose.yml", section.Path)
	assert.Equal(t, "/ws/proj", section.Dir)
	assert.True(t, section.Active)
	assert.Equal(t, &OutputWhen{Files: []string{"compose.yml"}, Here: true, Matched: "/ws/proj/compose.yml"}, section.When)

	items := map[string]OutputItem{}
	for _, item := range section.Cmds {
		items[item.Name] = item
	}
	assert.Len(t, items, 2, "compose:build does not match and does not list")
	assert.Equal(t, "compose:custom", items["custom"].ID)
	assert.True(t, items["custom"].Args)
	assert.Equal(t, []string{"docker", "compose"}, items["custom"].Exec, "the first exec: step the arguments follow")
	assert.Nil(t, items["shell"].Exec)
	assert.False(t, items["custom"].Interactive)
	assert.True(t, items["shell"].Interactive)
	assert.True(t, items["shell"].Active)

	all := buildListOutput(pipelines, OutputOptions{All: true})
	assert.Len(t, all, 3)
	var last OutputSection
	for _, section := range all {
		if section.ID == "composer" {
			last = section
		}
	}
	assert.Equal(t, "composer", last.ID)
	assert.False(t, last.Active)
	assert.Equal(t, "", last.Dir)
	assert.Equal(t, "/ws/.atkins/skills/composer.yml", last.Path)
	assert.Len(t, last.Cmds, 1)
	assert.False(t, last.Cmds[0].Active)

	// Without options the main pipeline runs in the folder of its file.
	plain := buildListOutput([]*model.Pipeline{mainPipeline})
	assert.Equal(t, "/ws/proj", plain[0].Dir)
}

// TestOutputError checks the lint failure output names the pipeline,
// its file and each finding, sorted by job.
func TestOutputError(t *testing.T) {
	t.Parallel()

	pipeline := &model.Pipeline{Name: "Broken", Path: "/ws/atkins.yml"}
	var result OutputError = NewOutputError(pipeline, []LintError{
		{Job: "default", Issue: "missing task reference", Detail: "no job b"},
	})
	assert.Equal(t, []OutputLintError{{Pipeline: "Broken", Path: "/ws/atkins.yml", Job: "default", Issue: "missing task reference", Detail: "no job b"}}, result.Lint)

	data, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"error": "pipeline 'Broken' has errors",
		"lint": [{"pipeline": "Broken", "path": "/ws/atkins.yml", "job": "default", "issue": "missing task reference", "detail": "no job b"}]
	}`, string(data))
}

// TestActivePipelines checks inactive skills drop out and a pipeline
// with an inactive job is a copy without it, the original untouched.
func TestActivePipelines(t *testing.T) {
	t.Parallel()

	compose := &model.Pipeline{ID: "compose", Jobs: map[string]*model.Job{
		"up":    {},
		"build": {Inactive: true},
	}}
	composer := &model.Pipeline{ID: "composer", Inactive: true, Jobs: map[string]*model.Job{"update": {}}}
	plain := &model.Pipeline{Tasks: map[string]*model.Job{"default": {}}}

	got := ActivePipelines([]*model.Pipeline{compose, composer, plain})
	assert.Len(t, got, 2)
	assert.Len(t, got[0].Jobs, 1)
	assert.NotNil(t, got[0].Jobs["up"])
	assert.Len(t, compose.Jobs, 2)
	assert.True(t, got[1] == plain)
}

// TestPrintJSON checks PrintJSON and PrintYAML write a value to stdout.
func TestPrintJSON(t *testing.T) {
	capture := func(fn func() error) string {
		old := os.Stdout
		r, w, err := os.Pipe()
		assert.NoError(t, err)
		os.Stdout = w

		assert.NoError(t, fn())
		assert.NoError(t, w.Close())
		os.Stdout = old

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		assert.NoError(t, err)
		return buf.String()
	}

	value := OutputError{Error: "e", Lint: []OutputLintError{}}
	assert.Equal(t, "{\n  \"error\": \"e\",\n  \"lint\": []\n}\n", capture(func() error { return PrintJSON(value) }))
	assert.Equal(t, "error: e\nlint: []\n", capture(func() error { return PrintYAML(value) }))
}

// TestStepLabels covers the labels a listing prints under a job, which
// a caller shows without reading the pipeline file itself.
func TestStepLabels(t *testing.T) {
	job := &model.Job{
		Steps: []*model.Step{
			{Task: "build"},
			{Run: "go test ./..."},
			{Desc: "Say hello", Run: "echo hello"},
			{Exec: []string{"go", "build", "./..."}},
			{},
		},
	}

	assert.Equal(t, []string{
		"task: build",
		"run: go test ./...",
		"Say hello",
		"exec: go build ./...",
	}, stepLabels(job))

	assert.Nil(t, stepLabels(&model.Job{}))
}

// TestIncludeFiles covers the files a declaration reads into its
// variables.
func TestIncludeFiles(t *testing.T) {
	assert.Nil(t, includeFiles(nil))
	assert.Nil(t, includeFiles(&model.Decl{}))
	assert.Equal(t, []string{"vars.yml"},
		includeFiles(&model.Decl{Include: &model.IncludeDecl{Files: []string{"vars.yml"}}}))
}

// TestEnvFiles covers the dotenv files a declaration reads into the
// environment.
func TestEnvFiles(t *testing.T) {
	assert.Nil(t, envFiles(nil))
	assert.Nil(t, envFiles(&model.Decl{}))
	assert.Nil(t, envFiles(&model.Decl{Env: &model.EnvDecl{}}))
	assert.Equal(t, []string{".env"},
		envFiles(&model.Decl{Env: &model.EnvDecl{Include: &model.IncludeDecl{Files: []string{".env"}}}}))
}

// TestBuildPipelineSectionCarriesTheWholeJob checks the fields a reader
// needs to list a folder's jobs without parsing the pipeline file: the
// job's name, what it does, what it needs and what it hides.
func TestBuildPipelineSectionCarriesTheWholeJob(t *testing.T) {
	hidden := false
	p := &model.Pipeline{
		Name: "Probe",
		Help: "What this pipeline is for.",
		Decl: &model.Decl{
			Include: &model.IncludeDecl{Files: []string{"vars/common.yml"}},
			Env:     &model.EnvDecl{Include: &model.IncludeDecl{Files: []string{".env.shared"}}},
		},
		Jobs: map[string]*model.Job{
			"default": {
				Desc:      "Run everything",
				Aliases:   []string{"all"},
				DependsOn: model.Dependencies{"build"},
				Steps:     []*model.Step{{Run: "echo done"}},
				Decl:      &model.Decl{Env: &model.EnvDecl{Include: &model.IncludeDecl{Files: []string{".env.job"}}}},
			},
			"hide": {Desc: "Hidden", Show: &hidden, Steps: []*model.Step{{Run: "echo hidden"}}},
		},
	}

	section := buildPipelineSection(p, "")
	assert.Equal(t, "Probe", section.Desc)
	assert.Equal(t, "What this pipeline is for.", section.Help)
	assert.Equal(t, []string{"vars/common.yml"}, section.Include)
	assert.Equal(t, []string{".env.shared"}, section.Env)

	byName := map[string]OutputItem{}
	for _, cmd := range section.Cmds {
		byName[cmd.Name] = cmd
	}

	assert.Equal(t, []string{"all"}, byName["default"].Aliases)
	assert.Equal(t, []string{"build"}, byName["default"].DependsOn)
	assert.Equal(t, []string{"run: echo done"}, byName["default"].Steps)
	assert.Equal(t, []string{".env.job"}, byName["default"].Env)
	assert.False(t, byName["default"].Hidden)

	// A job the file keeps out of menus stays in the listing, marked.
	// A caller knows it exists and decides for itself.
	assert.True(t, byName["hide"].Hidden)
}
