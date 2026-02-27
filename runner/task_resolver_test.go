package runner

import (
	"testing"

	"github.com/titpetric/atkins/model"
)

func TestTaskResolver_ResolveLocal(t *testing.T) {
	pipeline := &model.Pipeline{
		ID: "",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build the project"},
			"test":  {Desc: "Run tests"},
		},
	}

	resolver := &TaskResolver{
		CurrentPipeline: pipeline,
		AllPipelines:    []*model.Pipeline{pipeline},
	}

	resolved, err := resolver.Resolve("build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Name != "build" {
		t.Errorf("expected name 'build', got: %s", resolved.Name)
	}
	if resolved.Pipeline != pipeline {
		t.Errorf("expected pipeline to match")
	}
	if resolved.Job.Desc != "Build the project" {
		t.Errorf("expected job desc 'Build the project', got: %s", resolved.Job.Desc)
	}
}

func TestTaskResolver_ResolveLocalNotFound(t *testing.T) {
	pipeline := &model.Pipeline{
		ID:   "",
		Jobs: map[string]*model.Job{},
	}

	resolver := &TaskResolver{
		CurrentPipeline: pipeline,
		AllPipelines:    []*model.Pipeline{pipeline},
	}

	_, err := resolver.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != `task "nonexistent" not found in pipeline` {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestTaskResolver_ResolveCrossPipelineSkill(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID: "",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Main build"},
		},
	}
	goSkill := &model.Pipeline{
		ID: "go",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Go build"},
			"test":  {Desc: "Go test"},
		},
	}

	resolver := &TaskResolver{
		CurrentPipeline: mainPipeline,
		AllPipelines:    []*model.Pipeline{mainPipeline, goSkill},
	}

	// Resolve :go:build from main pipeline
	resolved, err := resolver.Resolve(":go:build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Name != "go:build" {
		t.Errorf("expected name 'go:build', got: %s", resolved.Name)
	}
	if resolved.Pipeline != goSkill {
		t.Errorf("expected pipeline to be goSkill")
	}
	if resolved.Job.Desc != "Go build" {
		t.Errorf("expected job desc 'Go build', got: %s", resolved.Job.Desc)
	}
}

func TestTaskResolver_ResolveCrossPipelineMain(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID: "",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Main build"},
		},
	}
	goSkill := &model.Pipeline{
		ID: "go",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Go build"},
		},
	}

	// When inside a skill, resolve :build to main pipeline
	resolver := &TaskResolver{
		CurrentPipeline: goSkill,
		AllPipelines:    []*model.Pipeline{mainPipeline, goSkill},
	}

	resolved, err := resolver.Resolve(":build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Name != "build" {
		t.Errorf("expected name 'build', got: %s", resolved.Name)
	}
	if resolved.Pipeline != mainPipeline {
		t.Errorf("expected pipeline to be mainPipeline")
	}
	if resolved.Job.Desc != "Main build" {
		t.Errorf("expected job desc 'Main build', got: %s", resolved.Job.Desc)
	}
}

func TestTaskResolver_ResolveSkillNotFound(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID:   "",
		Jobs: map[string]*model.Job{},
	}

	resolver := &TaskResolver{
		CurrentPipeline: mainPipeline,
		AllPipelines:    []*model.Pipeline{mainPipeline},
	}

	_, err := resolver.Resolve(":docker:build")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != `skill "docker" not found` {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestTaskResolver_ResolveTaskNotFoundInSkill(t *testing.T) {
	goSkill := &model.Pipeline{
		ID:   "go",
		Jobs: map[string]*model.Job{},
	}

	resolver := &TaskResolver{
		CurrentPipeline: goSkill,
		AllPipelines:    []*model.Pipeline{goSkill},
	}

	_, err := resolver.Resolve(":go:nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != `task "nonexistent" not found in skill "go"` {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestTaskResolver_ResolveMainPipelineNotFound(t *testing.T) {
	goSkill := &model.Pipeline{
		ID:   "go",
		Jobs: map[string]*model.Job{},
	}

	resolver := &TaskResolver{
		CurrentPipeline: goSkill,
		AllPipelines:    []*model.Pipeline{goSkill}, // No main pipeline
	}

	_, err := resolver.Resolve(":build")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "main pipeline not found" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestTaskResolver_ResolveFallbackWhenNoPipelines(t *testing.T) {
	pipeline := &model.Pipeline{
		ID: "",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build"},
		},
	}

	// When AllPipelines is empty, : prefix falls back to local lookup
	resolver := &TaskResolver{
		CurrentPipeline: pipeline,
		AllPipelines:    nil,
	}

	resolved, err := resolver.Resolve(":build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Name != "build" {
		t.Errorf("expected name 'build', got: %s", resolved.Name)
	}
}

func TestTaskResolver_ResolveWithTasks(t *testing.T) {
	// Test that Tasks field is used when Jobs is empty
	pipeline := &model.Pipeline{
		ID: "",
		Tasks: map[string]*model.Job{
			"build": {Desc: "Build task"},
		},
	}

	resolver := &TaskResolver{
		CurrentPipeline: pipeline,
		AllPipelines:    []*model.Pipeline{pipeline},
	}

	resolved, err := resolver.Resolve("build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Job.Desc != "Build task" {
		t.Errorf("expected job desc 'Build task', got: %s", resolved.Job.Desc)
	}
}

func TestTaskResolver_Validate(t *testing.T) {
	pipeline := &model.Pipeline{
		ID: "",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build"},
		},
	}

	resolver := &TaskResolver{
		CurrentPipeline: pipeline,
		AllPipelines:    []*model.Pipeline{pipeline},
	}

	// Valid task
	if err := resolver.Validate("build"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Invalid task
	if err := resolver.Validate("nonexistent"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestTaskResolver_ResolveSkillShorthand(t *testing.T) {
	// Test that "skill:job" without leading colon falls back to skill lookup
	mainPipeline := &model.Pipeline{
		ID:   "",
		Jobs: map[string]*model.Job{},
	}
	goSkill := &model.Pipeline{
		ID: "go",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Go build"},
		},
	}

	resolver := &TaskResolver{
		CurrentPipeline: mainPipeline,
		AllPipelines:    []*model.Pipeline{mainPipeline, goSkill},
	}

	// "go:build" should fall back to skill lookup when not found locally
	resolved, err := resolver.Resolve("go:build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Name != "go:build" {
		t.Errorf("expected name 'go:build', got: %s", resolved.Name)
	}
	if resolved.Pipeline != goSkill {
		t.Errorf("expected pipeline to be goSkill")
	}
	if resolved.Job.Desc != "Go build" {
		t.Errorf("expected job desc 'Go build', got: %s", resolved.Job.Desc)
	}
}

func TestTaskResolver_ResolveSkillShorthandLocalPreferred(t *testing.T) {
	// Test that local jobs take precedence over skill jobs with same colon-format name
	mainPipeline := &model.Pipeline{
		ID: "",
		Jobs: map[string]*model.Job{
			"go:build": {Desc: "Local go:build"},
		},
	}
	goSkill := &model.Pipeline{
		ID: "go",
		Jobs: map[string]*model.Job{
			"build": {Desc: "Go build"},
		},
	}

	resolver := &TaskResolver{
		CurrentPipeline: mainPipeline,
		AllPipelines:    []*model.Pipeline{mainPipeline, goSkill},
	}

	// "go:build" exists locally, so local version is preferred
	resolved, err := resolver.Resolve("go:build")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved.Pipeline != mainPipeline {
		t.Errorf("expected local pipeline to be preferred")
	}
	if resolved.Job.Desc != "Local go:build" {
		t.Errorf("expected job desc 'Local go:build', got: %s", resolved.Job.Desc)
	}
}

func TestGetJobsFromPipeline(t *testing.T) {
	t.Run("returns Jobs when present", func(t *testing.T) {
		p := &model.Pipeline{
			Jobs: map[string]*model.Job{
				"build": {},
			},
			Tasks: map[string]*model.Job{
				"test": {},
			},
		}
		jobs := getJobsFromPipeline(p)
		if _, ok := jobs["build"]; !ok {
			t.Error("expected 'build' job")
		}
		if _, ok := jobs["test"]; ok {
			t.Error("should not return tasks when jobs exist")
		}
	})

	t.Run("returns Tasks when Jobs is empty", func(t *testing.T) {
		p := &model.Pipeline{
			Jobs: map[string]*model.Job{},
			Tasks: map[string]*model.Job{
				"test": {},
			},
		}
		jobs := getJobsFromPipeline(p)
		if _, ok := jobs["test"]; !ok {
			t.Error("expected 'test' task")
		}
	})
}
