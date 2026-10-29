package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/titpetric/atkins/model"
)

// FindRepositoryRoot returns the folder holding .git, starting at
// startDir and traversing parent directories.
func FindRepositoryRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	for {
		// A worktree or a submodule has .git as a file.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository found above %s", startDir)
		}
		dir = parent
	}
}

// skillRefs returns the skill IDs a pipeline names through task: and
// depends_on:, in the order they are first seen.
func skillRefs(pipeline *model.Pipeline) []string {
	jobs := pipeline.GetJobs()
	seen := make(map[string]bool)

	var refs []string
	add := func(ref string) {
		id, ok := skillRefID(ref, jobs)
		if !ok || seen[id] {
			return
		}
		seen[id] = true
		refs = append(refs, id)
	}

	for _, job := range jobs {
		for _, dep := range job.DependsOn {
			add(dep)
		}
		for _, step := range job.Children() {
			if step.Task != "" {
				add(step.Task)
			}
		}
	}

	sort.Strings(refs)
	return refs
}

// skillRefID extracts the skill namespace from a task or dependency
// reference. A bare name is only a skill reference when the pipeline
// has no job of its own by that name.
func skillRefID(ref string, jobs map[string]*model.Job) (string, bool) {
	ref = strings.TrimPrefix(strings.TrimSpace(ref), ":")
	if ref == "" {
		return "", false
	}

	if index := strings.Index(ref, ":"); index > 0 {
		return ref[:index], true
	}

	if _, local := jobs[ref]; local {
		return "", false
	}

	return ref, true
}
