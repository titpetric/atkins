package runner

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"

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

	// First section should be main pipeline
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

	// Second section should be aliases
	if output[1].Desc != "Aliases" {
		t.Errorf("expected second section to be 'Aliases', got %s", output[1].Desc)
	}

	// Third section should be go skill
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

	// Check aliases are sorted
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
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ListPipelinesJSON([]*model.Pipeline{mainPipeline})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify it's valid JSON
	var parsed []ListOutputSection
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, output)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 section, got %d", len(parsed))
	}
	if parsed[0].Desc != "Main" {
		t.Errorf("expected section desc 'Main', got %s", parsed[0].Desc)
	}
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
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ListPipelinesYAML([]*model.Pipeline{mainPipeline})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify it's valid YAML
	var parsed []ListOutputSection
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\nOutput: %s", err, output)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 section, got %d", len(parsed))
	}
	if parsed[0].Desc != "Main" {
		t.Errorf("expected section desc 'Main', got %s", parsed[0].Desc)
	}
}
