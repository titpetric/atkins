package runner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/model"
)

// evaluateWhen checks if a skill's when: condition matches.
func (l *SkillsLoader) evaluateWhen(pipeline *model.Pipeline, workspaceDir string) (workDir, matched string, enabled bool) {
	// No when: condition means always enabled, use workspace dir
	if pipeline.When.IsEmpty() {
		return workspaceDir, "", true
	}

	match, found := MatchWhen(pipeline.When, l.StartDir)
	if !found {
		return "", "", false
	}

	return match.Dir, match.File, true
}

// FindFolder returns the nearest folder from startDir up that holds a folder named name.
func (l *SkillsLoader) FindFolder(name, startDir string) (containingDir string, found bool) {
	current := startDir
	for {
		candidate := filepath.Join(current, name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", false
		}
		current = parent
	}
}

// FindFile returns the nearest folder from startDir up where any pattern matches.
// It checks every pattern in a folder before moving to the parent.
func (l *SkillsLoader) FindFile(patterns []string, startDir string) (matchDir string, found bool) {
	// First, check absolute paths (no traversal needed)
	for _, pattern := range patterns {
		if filepath.IsAbs(pattern) {
			if match, ok := MatchWhenPattern("", pattern); ok {
				return filepath.Dir(match), true
			}
		}
	}

	// Search relative patterns from startDir going up
	current := startDir
	for {
		for _, pattern := range patterns {
			if filepath.IsAbs(pattern) {
				continue // Already handled above
			}

			if _, ok := MatchWhenPattern(current, pattern); ok {
				return current, true
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", false
		}
		current = parent
	}
}

// MatchWhenPattern returns what a `when: files:` pattern resolves to inside dir.
// A glob pattern such as `schema/*.up.sql` matches a folder's contents, `schema/` the folder.
func MatchWhenPattern(dir, pattern string) (match string, found bool) {
	candidate := filepath.Join(dir, pattern)

	if !strings.ContainsAny(pattern, "*?[") {
		if _, err := os.Stat(candidate); err != nil {
			return "", false
		}
		return candidate, true
	}

	// Glob reports only a malformed pattern as an error, which is the
	// skill author's problem to see as "never matches".
	matches, err := filepath.Glob(candidate)
	if err != nil || len(matches) == 0 {
		return "", false
	}

	return matches[0], true
}
