package config

import (
	"bytes"
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

// Default returns the embedded configuration.
//
// It is the base every other layer is applied to, and the source of the
// value substituted for any field left empty.
func Default() (*Config, error) {
	config := &Config{}
	if err := yaml.Unmarshal(RuntimeDefaultConfig, config); err != nil {
		return nil, fmt.Errorf("parse embedded config: %w", err)
	}
	return config, nil
}

// Load reads the effective configuration for a directory.
//
// The layers, each overriding the last: the embedded default, the user
// document at $HOME/.atkins/config.yml, the project document found by
// walking up from dir, and finally the environment overlay. The result
// is validated, which fills anything still empty.
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

// Paths returns the configuration documents that apply to dir, in the
// order they should be merged.
func Paths(dir string) []string {
	var paths []string

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, Dir, FileName))
	}

	if project := Discover(dir); project != "" {
		// A user document that is also the project document would
		// otherwise be merged twice; harmless, but confusing in a
		// --config listing.
		if len(paths) == 0 || paths[0] != project {
			paths = append(paths, project)
		}
	}

	return paths
}

// Discover walks up from dir looking for .atkins/config.yml, so a
// command run deep inside a project still finds the project's
// configuration.
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
// document that exists, or the path one would be created at.
func ProjectPath(dir string) string {
	if found := Discover(dir); found != "" {
		return found
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return filepath.Join(dir, Dir, FileName)
}

// mergeFile applies one document over the config. A missing file is
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
	// document doesn't mention alone, which is what makes these
	// layers rather than replacements.
	if err := yaml.Unmarshal(contents, c); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	return nil
}

// LoadFile reads a single document, with defaults and validation but
// without the user layer or the environment overlay. It is what the
// configuration menu edits.
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

// Save writes the configuration, creating the .atkins directory.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}

	encoded, err := c.Encode()
	if err != nil {
		return err
	}

	// The document can carry a signing key and an agent token, so it
	// is written for its owner only.
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// Encode renders the configuration as YAML.
func (c *Config) Encode() ([]byte, error) {
	var buffer bytes.Buffer

	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)

	if err := encoder.Encode(c); err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}

	return buffer.Bytes(), nil
}
