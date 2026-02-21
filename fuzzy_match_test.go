package main

import (
	"testing"

	"github.com/titpetric/atkins/model"
)

func TestFuzzyMatchError(t *testing.T) {
	t.Run("Error_Message", func(t *testing.T) {
		err := &FuzzyMatchError{Matches: []FuzzyMatch{
			{JobName: "test1"},
			{JobName: "test2"},
		}}
		expected := "multiple jobs match the pattern; use -l to see all jobs"
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})
}

func TestFindFuzzyMatches(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		pipelines []*model.Pipeline
		want      []string // FullNames of expected matches
	}{
		{
			name:    "exact_job_match",
			pattern: "push",
			pipelines: []*model.Pipeline{
				{
					ID: "docker",
					Jobs: map[string]*model.Job{
						"push":  {Name: "Push Docker image"},
						"build": {Name: "Build Docker image"},
					},
				},
			},
			want: []string{"docker:push"},
		},
		{
			name:    "substring_match_single_result",
			pattern: "mergecov",
			pipelines: []*model.Pipeline{
				{
					ID: "",
					Jobs: map[string]*model.Job{
						"test:mergecov": {Name: "Merge coverage"},
						"test:simple":   {Name: "Simple test"},
					},
				},
			},
			want: []string{"test:mergecov"},
		},
		{
			name:    "multiple_matches",
			pattern: "test",
			pipelines: []*model.Pipeline{
				{
					ID: "go",
					Jobs: map[string]*model.Job{
						"test":  {Name: "Run tests"},
						"build": {Name: "Build"},
					},
				},
				{
					ID: "docker",
					Jobs: map[string]*model.Job{
						"test": {Name: "Test Docker image"},
						"push": {Name: "Push Docker image"},
					},
				},
			},
			want: []string{"go:test", "docker:test"},
		},
		{
			name:    "case_insensitive_match",
			pattern: "PUSH",
			pipelines: []*model.Pipeline{
				{
					ID: "docker",
					Jobs: map[string]*model.Job{
						"push":  {Name: "Push Docker image"},
						"build": {Name: "Build Docker image"},
					},
				},
			},
			want: []string{"docker:push"},
		},
		{
			name:    "suffix_match",
			pattern: "build",
			pipelines: []*model.Pipeline{
				{
					ID: "go",
					Jobs: map[string]*model.Job{
						"build": {Name: "Build app"},
					},
				},
				{
					ID: "docker",
					Jobs: map[string]*model.Job{
						"build": {Name: "Build image"},
					},
				},
			},
			want: []string{"go:build", "docker:build"},
		},
		{
			name:    "no_matches",
			pattern: "nonexistent",
			pipelines: []*model.Pipeline{
				{
					ID: "go",
					Jobs: map[string]*model.Job{
						"test":  {Name: "Run tests"},
						"build": {Name: "Build"},
					},
				},
			},
			want: []string{},
		},
		{
			name:    "main_pipeline_job_match",
			pattern: "fmt",
			pipelines: []*model.Pipeline{
				{
					ID: "",
					Jobs: map[string]*model.Job{
						"fmt":  {Name: "Format code"},
						"test": {Name: "Test"},
					},
				},
			},
			want: []string{"fmt"},
		},
		{
			name:    "partial_substring_match",
			pattern: "pub",
			pipelines: []*model.Pipeline{
				{
					ID: "release",
					Jobs: map[string]*model.Job{
						"publish": {Name: "Publish"},
						"build":   {Name: "Build"},
					},
				},
			},
			want: []string{"release:publish"},
		},
		{
			name:    "empty_pipeline",
			pattern: "test",
			pipelines: []*model.Pipeline{
				{
					ID:   "empty",
					Jobs: map[string]*model.Job{},
				},
			},
			want: []string{},
		},
		{
			name:    "tasks_fallback",
			pattern: "task1",
			pipelines: []*model.Pipeline{
				{
					ID: "skill",
					Tasks: map[string]*model.Job{
						"task1": {Name: "Task 1"},
						"task2": {Name: "Task 2"},
					},
				},
			},
			want: []string{"skill:task1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFuzzyMatches(tt.pipelines, tt.pattern)

			if len(got) != len(tt.want) {
				t.Errorf("got %d matches, want %d", len(got), len(tt.want))
				return
			}

			// Check if all expected matches are present
			matchMap := make(map[string]bool)
			for _, match := range got {
				matchMap[match.FullName] = true
			}

			for _, want := range tt.want {
				if !matchMap[want] {
					t.Errorf("expected match %q not found in results", want)
				}
			}
		})
	}
}

func TestFuzzyMatch_StructFields(t *testing.T) {
	t.Run("Pipeline_pointer_set", func(t *testing.T) {
		pipeline := &model.Pipeline{ID: "test"}
		match := FuzzyMatch{
			Pipeline: pipeline,
			JobName:  "job",
			FullName: "test:job",
		}
		if match.Pipeline != pipeline {
			t.Error("Pipeline pointer not set correctly")
		}
	})

	t.Run("JobName_set", func(t *testing.T) {
		match := FuzzyMatch{JobName: "myjob"}
		if match.JobName != "myjob" {
			t.Errorf("got %q, want %q", match.JobName, "myjob")
		}
	})

	t.Run("FullName_with_skill", func(t *testing.T) {
		match := FuzzyMatch{FullName: "skill:job"}
		if match.FullName != "skill:job" {
			t.Errorf("got %q, want %q", match.FullName, "skill:job")
		}
	})

	t.Run("FullName_without_skill", func(t *testing.T) {
		match := FuzzyMatch{FullName: "job"}
		if match.FullName != "job" {
			t.Errorf("got %q, want %q", match.FullName, "job")
		}
	})
}
