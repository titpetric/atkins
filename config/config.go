package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Version is the document version this build understands.
const Version = 1

// Config is the .atkins/config.yml document. A value holds the layers
// merged so far; Load and LoadFile return one that validates, and Save
// writes it back.
type Config struct {
	// Version identifies the document format.
	Version int `yaml:"version" json:"version"`

	// Storage is what a run records and where.
	Storage StorageConfig `yaml:"storage" json:"storage"`
}

// StorageConfig is the event database: the file every run appends a row
// per job and per command to, and what it holds.
type StorageConfig struct {
	// Database is the workspace's file in place of
	// $HOME/.atkins/atkins.db, or a list of files in the form of PATH:
	// the first is the workspace's, the rest are mirrors that take the
	// same rows and are never read. The project's, the atkins.db of the
	// nearest .atkins folder, records as well either way.
	Database string `yaml:"database" json:"database" env:"ATKINS_DB"`

	// Record writes a row per job and per command of every run. Off
	// records nothing, the same as `--local`, and leaves the database
	// alone.
	//
	// Retention is no setting: storage.Retention is 31 days on every
	// machine, applied by the run that records.
	Record bool `yaml:"record" json:"record" env:"ATKINS_RECORD"`
}

// Default returns the embedded configuration, or the error parsing it.
func Default() (*Config, error) {
	config := &Config{}
	if err := yaml.Unmarshal(RuntimeDefaultConfig, config); err != nil {
		return nil, fmt.Errorf("parse embedded config: %w", err)
	}
	return config, nil
}

// ApplyEnvironment overlays the non-empty ATKINS_* variables of environ.
func (c *Config) ApplyEnvironment(environ []string) {
	values := make(map[string]string, len(environ))
	for _, entry := range environ {
		name, value, found := strings.Cut(entry, "=")
		if found && value != "" {
			values[name] = value
		}
	}

	if len(values) == 0 {
		return
	}

	applyEnvironment(reflect.ValueOf(c).Elem(), values)
}

// Save writes the configuration to path, owner-only, creating its directory.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}

	encoded, err := c.Encode()
	if err != nil {
		return err
	}

	// The document can carry a signing key and an agent token, so Save
	// writes it for its owner only.
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// Encode renders the configuration as YAML indented by two spaces.
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
