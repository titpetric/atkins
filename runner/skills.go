package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/model"
)

// Skills handles loading skill pipelines from disk directories.
type Skills struct {
	Dirs    []string // Directories to search (in priority order)
	WorkDir string   // Directory used for when.files checks (defaults to cwd if empty)
}

// NewSkills will create a new skills loader.
// If jailed it only searches `.atkins/skills/` in project root.
// If not jailed, it also loads `$HOME/.atkins/skills/`.
func NewSkills(projectRoot string, jail bool) *Skills {
	dirs := []string{
		filepath.Join(projectRoot, ".atkins", "skills"),
	}
	if !jail {
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, ".atkins", "skills"))
		}
	}
	return &Skills{Dirs: dirs}
}

// NewGlobalSkills creates a skills loader that only searches $HOME/.atkins/skills/.
func NewGlobalSkills() *Skills {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".atkins", "skills"))
	}
	return &Skills{Dirs: dirs}
}

// Load discovers and returns all skill pipelines that match their When conditions.
func (s *Skills) Load() ([]*model.Pipeline, error) {
	return discoverSkillsFromDirs(s.Dirs, s.WorkDir)
}

// interpolateString expands shell commands in $(command) syntax.
func interpolateString(s string) (string, error) {
	// Handle $(command) syntax
	for strings.Contains(s, "$(") {
		start := strings.Index(s, "$(")
		end := strings.Index(s[start:], ")")
		if end == -1 {
			break
		}
		end += start

		cmd := s[start+2 : end]
		output, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			return "", fmt.Errorf("failed to execute command %q: %w", cmd, err)
		}

		result := strings.TrimSpace(string(output))
		s = s[:start] + result + s[end+1:]
	}

	return s, nil
}

// findFileInAncestors checks if a file exists at the given path or in parent directories.
// For absolute paths, checks directly. For relative paths, searches upward from startDir
// (or cwd if startDir is empty) up to (but not including) the filesystem root.
func findFileInAncestors(path string, startDir string) bool {
	if filepath.IsAbs(path) {
		_, err := os.Stat(path)
		return err == nil
	}

	// Relative path: search from startDir (or cwd) upward
	current := startDir
	if current == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return false
		}
		current = cwd
	}
	for {
		testPath := filepath.Join(current, path)
		if _, err := os.Stat(testPath); err == nil {
			return true
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}

	return false
}

// discoverSkillsFromDirs loads all skill YAML files from the given directories
// and returns pipelines that match their When conditions.
// workDir is used as the starting directory for when.files checks (cwd if empty).
func discoverSkillsFromDirs(dirs []string, workDir string) ([]*model.Pipeline, error) {
	var skillPipelines []*model.Pipeline

	// Track seen files to avoid duplicates (project-local takes precedence)
	seen := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
				continue
			}

			if seen[entry.Name()] {
				continue // Skip if already loaded from a higher-priority dir
			}
			seen[entry.Name()] = true

			path := filepath.Join(dir, entry.Name())
			pipeline, err := loadSkillFromFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to load skill from %s: %w", path, err)
			}

			// Check if this skill's When conditions are met
			if pipeline.When != nil && len(pipeline.When.Files) > 0 {
				enabled := false
				for _, pattern := range pipeline.When.Files {
					// Interpolate the pattern
					interpolated, err := interpolateString(pattern)
					if err != nil {
						return nil, fmt.Errorf("failed to interpolate pattern %q in skill %s: %w", pattern, pipeline.Name, err)
					}

					// Check if file exists (relative paths search up parent dirs)
					if findFileInAncestors(interpolated, workDir) {
						enabled = true
						break // At least one pattern matched
					}
				}

				if !enabled {
					continue // Skip this skill if no patterns matched
				}
			}

			skillPipelines = append(skillPipelines, pipeline)
		}
	}

	return skillPipelines, nil
}

// loadSkillFromFile loads a skill pipeline from a file on disk.
// Sets Pipeline.ID from the filename (e.g., "go.yml" -> "go").
func loadSkillFromFile(path string) (*model.Pipeline, error) {
	pipelines, err := LoadPipeline(path)
	if err != nil {
		return nil, err
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipeline found in skill file %s", path)
	}

	// Set ID from filename without extension
	pipeline := pipelines[0]
	filename := filepath.Base(path)
	pipeline.ID = strings.TrimSuffix(filename, filepath.Ext(filename))

	return pipeline, nil
}
