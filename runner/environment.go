package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment represents the discovered project environment.
type Environment struct {
	Root string // Project root directory
}

// projectMarkers defines files/directories that indicate a project root.
var projectMarkers = []string{
	"go.mod",
	"Dockerfile",
	"compose.yml",
	"docker-compose.yml",
	".github/",
	"schema/",
}

// DiscoverEnvironment scans for marker files starting from startDir,
// traversing parent directories until the filesystem root is reached.
// Root is set to the highest directory that contains any markers.
func DiscoverEnvironment(startDir string) (*Environment, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	root := ""
	dir := absStart
	for {
		if hasProjectMarker(dir) {
			root = dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if root == "" {
		return nil, fmt.Errorf("no project root found (searched for go.mod, Dockerfile, compose.yml)")
	}

	return &Environment{Root: root}, nil
}

// DiscoverEnvironmentFromCwd is a convenience wrapper that starts from the current working directory.
func DiscoverEnvironmentFromCwd() (*Environment, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return DiscoverEnvironment(cwd)
}

// hasProjectMarker checks if any marker file/directory exists in dir.
func hasProjectMarker(dir string) bool {
	for _, marker := range projectMarkers {
		path := filepath.Join(dir, marker)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		// Markers ending with "/" expect a directory, others expect a file
		if strings.HasSuffix(marker, "/") {
			if info.IsDir() {
				return true
			}
		} else {
			if !info.IsDir() {
				return true
			}
		}
	}
	return false
}
