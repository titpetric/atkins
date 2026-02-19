package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment represents the discovered project environment.
type Environment struct {
	Root   string   // Project root directory
	Skills []string // List of activated skill names
}

// DiscoverEnvironment scans for marker files starting from startDir,
// traversing parent directories until the filesystem root is reached.
// Skills are accumulated from all levels; Root is set to the highest
// directory that contributed any markers (e.g. the go.mod location).
func DiscoverEnvironment(startDir string) (*Environment, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	seen := make(map[string]bool)
	var allSkills []string
	root := ""

	dir := absStart
	for {
		skills := detectSkills(dir)
		for _, s := range skills {
			if !seen[s] {
				seen[s] = true
				allSkills = append(allSkills, s)
			}
		}
		if len(skills) > 0 {
			root = dir // highest dir with markers becomes root
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if len(allSkills) == 0 {
		return nil, fmt.Errorf("no environment markers found (searched for go.mod, Dockerfile, compose.yml)")
	}

	return &Environment{
		Root:   root,
		Skills: allSkills,
	}, nil
}

// DiscoverEnvironmentFromCwd is a convenience wrapper that starts from the current working directory.
func DiscoverEnvironmentFromCwd() (*Environment, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return DiscoverEnvironment(cwd)
}

// detectSkills checks which marker files exist in the given directory.
// Returns marker file names without extensions (e.g., "go" for "go.mod").
// This was used for legacy skill detection but is superseded by Pipeline.When checks.
func detectSkills(dir string) []string {
	var skills []string

	// Check for standard project markers
	markers := map[string]string{
		"go.mod":             "go",
		"Dockerfile":         "docker",
		"compose.yml":        "compose",
		"docker-compose.yml": "compose",
		".github/":           "github",
		"schema/":            "mig",
	}

	for marker, skillName := range markers {
		path := filepath.Join(dir, marker)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		// Markers ending with "/" expect a directory, others expect a file
		if strings.HasSuffix(marker, "/") {
			if info.IsDir() {
				skills = append(skills, skillName)
			}
		} else {
			if !info.IsDir() {
				skills = append(skills, skillName)
			}
		}
	}

	return skills
}
