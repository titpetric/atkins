package runner

import (
	"errors"
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

func TestResolveJobTarget(t *testing.T) {
	// Helper to create a pipeline with jobs
	makePipeline := func(id string, jobs map[string]*model.Job) *model.Pipeline {
		return &model.Pipeline{ID: id, Jobs: jobs}
	}

	t.Run("invoked_pipeline_main", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"up": {}}),
			makePipeline("docker", map[string]*model.Job{"up": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget(":up")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("expected main pipeline (ID=''), got: %q", target.Pipeline.ID)
		}
		if target.JobName != "up" {
			t.Errorf("expected job 'up', got: %q", target.JobName)
		}
	})

	t.Run("invoked_pipeline_skill", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"up": {}}),
			makePipeline("docker", map[string]*model.Job{"build": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget(":docker:build")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "docker" {
			t.Errorf("expected pipeline 'docker', got: %q", target.Pipeline.ID)
		}
		if target.JobName != "build" {
			t.Errorf("expected job 'build', got: %q", target.JobName)
		}
	})

	t.Run("prefixed_job_reference", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"test": {}}),
			makePipeline("go", map[string]*model.Job{"test": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("go:test")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "go" {
			t.Errorf("expected pipeline 'go', got: %q", target.Pipeline.ID)
		}
		if target.JobName != "test" {
			t.Errorf("expected job 'test', got: %q", target.JobName)
		}
	})

	t.Run("main_pipeline_exact_match_over_alias", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"up": {}}),
			makePipeline("docker", map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("up")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("main pipeline should take precedence over alias, got pipeline: %q", target.Pipeline.ID)
		}
		if target.JobName != "up" {
			t.Errorf("expected job 'up', got: %q", target.JobName)
		}
	})

	t.Run("main_pipeline_exact_match_over_skill_alias", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("go", map[string]*model.Job{
				"compile": {Aliases: []string{"build"}},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("build")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("main pipeline exact match should precede alias, got pipeline: %q", target.Pipeline.ID)
		}
		if target.JobName != "build" {
			t.Errorf("expected job 'build', got: %q", target.JobName)
		}
	})

	t.Run("alias_match_when_no_main_pipeline_job", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("up")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "docker" {
			t.Errorf("alias should match when no main pipeline job exists, got pipeline: %q", target.Pipeline.ID)
		}
		if target.JobName != "start" {
			t.Errorf("expected job 'start', got: %q", target.JobName)
		}
	})

	t.Run("main_pipeline_alias_over_skill_exact_match", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{
				"build": {Aliases: []string{"default"}},
			}),
			makePipeline("go", map[string]*model.Job{
				"default": {},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("default")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("main pipeline alias should take precedence over skill job name, got pipeline: %q", target.Pipeline.ID)
		}
		if target.JobName != "build" {
			t.Errorf("expected job 'build', got: %q", target.JobName)
		}
	})

	t.Run("skill_name_without_job_errors", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{"build": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		_, err := resolver.ResolveJobTarget("docker")
		if err == nil {
			t.Fatal("skill name alone should error, use docker:job syntax")
		}
		if !contains(err.Error(), "not found") {
			t.Errorf("expected error containing 'not found', got: %q", err.Error())
		}
	})

	t.Run("fuzzy_match_single_result", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{
				"test:mergecov": {},
				"test:simple":   {},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("mergecov")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("expected main pipeline, got: %q", target.Pipeline.ID)
		}
		if target.JobName != "test:mergecov" {
			t.Errorf("expected job 'test:mergecov', got: %q", target.JobName)
		}
	})

	t.Run("exact_match_in_skill_pipelines", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("go", map[string]*model.Job{"test": {}}),
			makePipeline("python", map[string]*model.Job{"test": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("test")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "go" {
			t.Errorf("first skill with exact match wins, got: %q", target.Pipeline.ID)
		}
		if target.JobName != "test" {
			t.Errorf("expected job 'test', got: %q", target.JobName)
		}
	})

	t.Run("fuzzy_match_multiple_results_error", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("tools", map[string]*model.Job{
				"go-build":     {},
				"docker-build": {},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		_, err := resolver.ResolveJobTarget("build")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fuzzyErr *FuzzyMatchError
		if !errAs(err, &fuzzyErr) {
			t.Fatalf("expected FuzzyMatchError, got: %T", err)
		}
		if len(fuzzyErr.Matches) != 2 {
			t.Errorf("expected 2 matches, got: %d", len(fuzzyErr.Matches))
		}
	})

	t.Run("error_on_not_found", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{"push": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		_, err := resolver.ResolveJobTarget("nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !contains(err.Error(), "not found") {
			t.Errorf("expected error containing 'not found', got: %q", err.Error())
		}
	})

	t.Run("colon_job_name_not_treated_as_skill_prefix", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{
				"test:mergecov": {},
				"test:simple":   {},
				"default":       {},
			}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("test:mergecov")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("should match main pipeline, not look for skill 'test', got: %q", target.Pipeline.ID)
		}
		if target.JobName != "test:mergecov" {
			t.Errorf("expected job 'test:mergecov', got: %q", target.JobName)
		}
	})

	t.Run("colon_job_prefers_exact_over_skill", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"go:test": {}}),
			makePipeline("go", map[string]*model.Job{"test": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("go:test")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("exact main pipeline match should win over skill prefix, got: %q", target.Pipeline.ID)
		}
		if target.JobName != "go:test" {
			t.Errorf("expected job 'go:test', got: %q", target.JobName)
		}
	})

	t.Run("error_job_not_found_with_colon", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		_, err := resolver.ResolveJobTarget("nonexistent:job")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !contains(err.Error(), "not found") {
			t.Errorf("expected error containing 'not found', got: %q", err.Error())
		}
	})

	t.Run("tasks_field_supported", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			{ID: "", Tasks: map[string]*model.Job{"up": {}}},
			{ID: "docker", Tasks: map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}},
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("up")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("main pipeline Tasks should take precedence, got pipeline: %q", target.Pipeline.ID)
		}
		if target.JobName != "up" {
			t.Errorf("expected job 'up', got: %q", target.JobName)
		}
	})

	t.Run("main_pipeline_order_independent", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("docker", map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}),
			makePipeline("", map[string]*model.Job{"up": {}}),
		}
		resolver := &TaskResolver{AllPipelines: pipelines}
		target, err := resolver.ResolveJobTarget("up")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if target.Pipeline.ID != "" {
			t.Errorf("main pipeline exact match should work regardless of order, got pipeline: %q", target.Pipeline.ID)
		}
		if target.JobName != "up" {
			t.Errorf("expected job 'up', got: %q", target.JobName)
		}
	})
}

// contains is a helper for checking substrings in test assertions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// errAs is a helper wrapping errors.As for test use.
func errAs(err error, target **FuzzyMatchError) bool {
	return errors.As(err, target)
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
		jobs := getJobs(p)
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
		jobs := getJobs(p)
		if _, ok := jobs["test"]; !ok {
			t.Error("expected 'test' task")
		}
	})
}
