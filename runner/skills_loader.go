package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/model"
)

// SkillsLoader discovers and loads skill pipelines from .atkins/skills/ directories.
// It evaluates `when:` conditions to determine which skills are enabled and sets
// the appropriate working directory for each skill based on the rules.
type SkillsLoader struct {
	// SkillsDirs are the directories to search for skill files, in priority order.
	// First directory takes precedence for skills with the same ID.
	SkillsDirs []string

	// StartDir is the directory from which to start searching for when: files.
	// This is typically the user's working directory.
	StartDir string

	// WorkspaceDir is the folder containing .atkins/ (used for skills without when:).
	WorkspaceDir string
}

// NewSkillsLoader creates a loader for the given workspace.
// workspaceDir is the folder containing .atkins/ (used as Dir for skills without when:).
// startDir is where to start searching for when: files (typically user's cwd).
func NewSkillsLoader(workspaceDir, startDir string) *SkillsLoader {
	return &SkillsLoader{
		SkillsDirs:   []string{filepath.Join(workspaceDir, ".atkins", "skills")},
		StartDir:     startDir,
		WorkspaceDir: workspaceDir,
	}
}

// AddSkillsDir adds an additional skills directory to search.
// Directories added later have lower precedence.
func (l *SkillsLoader) AddSkillsDir(dir string) {
	l.SkillsDirs = append(l.SkillsDirs, dir)
}

// Load discovers and returns all enabled skill pipelines.
func (l *SkillsLoader) Load() ([]*model.Pipeline, error) {
	var pipelines []*model.Pipeline
	seen := make(map[string]bool) // Track skill IDs for deduplication

	for _, skillsDir := range l.SkillsDirs {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read skills directory %s: %w", skillsDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
				continue
			}

			// Load the skill file
			skillPath := filepath.Join(skillsDir, entry.Name())
			pipeline, err := l.loadSkillFile(skillPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load skill %s: %w", skillPath, err)
			}

			// Skip if already loaded from higher-priority directory
			if seen[pipeline.ID] {
				continue
			}

			// Evaluate when: condition and determine working directory
			workDir, enabled := l.evaluateWhen(pipeline)
			if !enabled {
				continue
			}

			// Set Dir only if not already explicitly set in the skill file
			if pipeline.Dir == "" {
				pipeline.Dir = workDir
			}

			seen[pipeline.ID] = true
			pipelines = append(pipelines, pipeline)
		}
	}

	return pipelines, nil
}

// loadSkillFile loads a single skill pipeline from a YAML file.
// Sets Pipeline.ID from the filename (e.g., "go.yml" → "go").
func (l *SkillsLoader) loadSkillFile(path string) (*model.Pipeline, error) {
	pipelines, err := LoadPipeline(path)
	if err != nil {
		return nil, err
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipeline found in skill file %s", path)
	}

	pipeline := pipelines[0]
	filename := filepath.Base(path)
	pipeline.ID = strings.TrimSuffix(filename, filepath.Ext(filename))

	return pipeline, nil
}

// evaluateWhen checks if a skill's when: condition is satisfied.
func (l *SkillsLoader) evaluateWhen(pipeline *model.Pipeline) (workDir string, enabled bool) {
	// No when: condition means always enabled, use workspace dir
	if pipeline.When == nil || len(pipeline.When.Files) == 0 {
		return l.WorkspaceDir, true
	}

	// Find the first matching file from any pattern
	matchDir, found := l.FindFile(pipeline.When.Files, l.StartDir)
	if !found {
		return "", false
	}

	return matchDir, true
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
// For each directory (starting with startDir, going up), all patterns are checked.
// This means closer matches are preferred over pattern order.
func (l *SkillsLoader) FindFile(patterns []string, startDir string) (matchDir string, found bool) {
	// First, check absolute paths (no traversal needed)
	for _, pattern := range patterns {
		if filepath.IsAbs(pattern) {
			if _, err := os.Stat(pattern); err == nil {
				return filepath.Dir(pattern), true
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

			candidate := filepath.Join(current, pattern)
			if _, err := os.Stat(candidate); err == nil {
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
