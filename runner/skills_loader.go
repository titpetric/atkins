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
//
// A skill from an earlier directory shadows one of the same ID found
// later, so a project skill wins over the global one it shares a name
// with. A skill whose when: block does not match contributes nothing and
// does not shadow anything.
func (l *SkillsLoader) Load() ([]*model.Pipeline, error) {
	skills, err := l.Scan()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)

	var pipelines []*model.Pipeline
	for _, skill := range skills {
		if !skill.Active || seen[skill.ID()] {
			continue
		}
		seen[skill.ID()] = true
		pipelines = append(pipelines, skill.Pipeline)
	}

	return pipelines, nil
}

// Scan returns every skill file in the loader's directories, in
// directory order, whether or not its when: block is satisfied.
//
// Load drops the skills that do not apply here; Scan keeps them so
// `atkins --help` can list what is installed rather than only what the
// current directory activates. Two directories carrying the same skill
// ID both appear, the higher-priority one first.
func (l *SkillsLoader) Scan() ([]*Skill, error) {
	var skills []*Skill

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

			skillPath := filepath.Join(skillsDir, entry.Name())
			skill, err := l.loadSkill(skillPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load skill %s: %w", skillPath, err)
			}

			skill.Dir, skill.Active = l.evaluateWhen(skill.Pipeline)

			// Set Dir only if not already explicitly set in the skill file
			if skill.Active && skill.Pipeline.Dir == "" {
				skill.Pipeline.Dir = skill.Dir
			}

			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// loadSkill reads a skill file and the optional markdown companion
// beside it: go.yml is documented by go.md in the same directory.
// A missing or unreadable companion leaves Doc empty and is not an
// error, because the guide is optional.
func (l *SkillsLoader) loadSkill(path string) (*Skill, error) {
	pipeline, err := l.loadSkillFile(path)
	if err != nil {
		return nil, err
	}

	skill := &Skill{Pipeline: pipeline, Path: path}

	docPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".md"
	if data, err := os.ReadFile(docPath); err == nil {
		skill.DocPath = docPath
		skill.Doc = strings.TrimSpace(string(data))
	}

	return skill, nil
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
// A pattern carrying a glob meta character is expanded, so a skill can
// activate on the contents of a folder rather than its name:
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
