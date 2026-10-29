package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

// FileName is the configuration document inside an .atkins directory.
const FileName = "config.yml"

// Dir is the directory holding project and user configuration.
const Dir = ".atkins"

// Load returns the merged and validated configuration for a directory,
// or the error of the default, a document or validation.
func Load(dir string) (*Config, error) {
	config, err := Default()
	if err != nil {
		return nil, err
	}

	for _, path := range Paths(dir) {
		if err := config.mergeFile(path); err != nil {
			return nil, err
		}
	}

	config.ApplyEnvironment(os.Environ())

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Paths returns the documents that apply to dir, in the order Load merges them.
func Paths(dir string) []string {
	var paths []string

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, Dir, FileName))
	}

	if project := Discover(dir); project != "" {
		// A user document that is also the project document would
		// otherwise merge twice; harmless, but confusing in the
		// configuration menu's listing.
		if len(paths) == 0 || paths[0] != project {
			paths = append(paths, project)
		}
	}

	return paths
}

// Discover walks up from dir looking for .atkins/config.yml. A command
// run deep inside a project finds the project's configuration.
func Discover(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	current, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(current, Dir, FileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// ProjectPath returns where a project's configuration belongs: the
// document that exists, or the path where a new one belongs.
func ProjectPath(dir string) string {
	if found := Discover(dir); found != "" {
		return found
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return filepath.Join(dir, Dir, FileName)
}

// mergeFile applies one document over the configuration. A missing file is
// not an error: the layer is simply absent.
func (c *Config) mergeFile(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	// Unmarshalling over the existing struct leaves fields the
	// document doesn't mention alone. Each document is a layer
	// and never a replacement.
	if err := yaml.Unmarshal(contents, c); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	return nil
}

// LoadFile reads a single document, with defaults and validation but
// without the user layer or the environment overlay. The
// configuration menu edits this result.
func LoadFile(path string) (*Config, error) {
	config, err := Default()
	if err != nil {
		return nil, err
	}
	if err := config.mergeFile(path); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
