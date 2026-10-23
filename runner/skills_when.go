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

// FindFolder searches for a directory with the given name starting from startDir
// and traversing parent directories. Returns (found, containingDir) where
// containingDir is the parent directory that contains the named folder.
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

// FindFile searches for files matching any of the given patterns starting from
// startDir and traversing parent directories. Returns (found, matchDir) where
// matchDir is the directory containing the first matched file.
//
// For each directory (starting with startDir, going up), it checks all patterns.
// This means closer matches win over pattern order.
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

// MatchWhenPattern reports whether a `when: files:` pattern resolves to
// something inside dir, and what it resolved to.
//
// MatchWhenPattern expands a pattern carrying a glob meta character, so a
// skill can activate on the contents of a folder and not only its name:
// `schema/*.up.sql` matches a folder of migrations, while `schema/`
// matches any folder of that name.
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
