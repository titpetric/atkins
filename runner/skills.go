package runner

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/model"
)

// SkillDefinition maps a skill name to its trigger conditions.
type SkillDefinition struct {
	Name    string   // Skill name (e.g., "go", "docker", "compose")
	File    string   // YAML filename (e.g., "go.yml")
	Markers []string // Files that trigger this skill (e.g., ["go.mod"])
}

// SkillsDirs returns the directories to scan for skill files, in priority order.
// Project-local (.atkins/skills/) takes precedence over user-level ($HOME/.atkins/skills/).
func SkillsDirs(projectRoot string) []string {
	dirs := []string{
		filepath.Join(projectRoot, ".atkins", "skills"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".atkins", "skills"))
	}
	return dirs
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
// For absolute paths, checks directly. For relative paths, searches upward from cwd
// up to (but not including) the filesystem root.
func findFileInAncestors(path string) bool {
	if filepath.IsAbs(path) {
		_, err := os.Stat(path)
		return err == nil
	}

	// Relative path: search from cwd upward
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	current := cwd
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

// DiscoverSkillsFromDirs loads all skill YAML files from the given directories
// and returns pipelines that match their When conditions.
func DiscoverSkillsFromDirs(dirs []string) ([]*model.Pipeline, error) {
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
			pipeline, err := LoadSkillFromFile(path)
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
					if findFileInAncestors(interpolated) {
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

// FindSkillFile searches disk directories for a skill file, returning the path if found.
func FindSkillFile(dirs []string, filename string) string {
	for _, dir := range dirs {
		path := filepath.Join(dir, filename)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// LoadSkillFromFile loads a skill pipeline from a file on disk.
func LoadSkillFromFile(path string) (*model.Pipeline, error) {
	pipelines, err := LoadPipeline(path)
	if err != nil {
		return nil, err
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipeline found in skill file %s", path)
	}
	return pipelines[0], nil
}

// LoadSkillFromFS loads a skill pipeline from an embedded filesystem.
func LoadSkillFromFS(skillsFS fs.FS, filename string) (*model.Pipeline, error) {
	data, err := fs.ReadFile(skillsFS, "skills/"+filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded skill %s: %w", filename, err)
	}

	pipelines, err := LoadPipelineFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded skill %s: %w", filename, err)
	}

	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipeline found in embedded skill %s", filename)
	}

	return pipelines[0], nil
}
