package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/model"
)

// LoadPipeline loads and parses a pipeline from a yaml file.
// Returns the number of documents loaded, the parsed pipeline, and any error.
func LoadPipeline(filePath string) ([]*model.Pipeline, error) {
	// Read the raw file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	pipelines, err := LoadPipelineFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	// Set default name from filename if not specified
	if pipelines[0].Name == "" {
		pipelines[0].Name = filepath.Base(filePath)
	}

	return pipelines, nil
}

// LoadPipelineFromReader loads and parses a pipeline from an io.Reader.
// Returns the parsed pipeline(s) and any error.
func LoadPipelineFromReader(r io.Reader) ([]*model.Pipeline, error) {
	// Parse with plain YAML first (no expression evaluation)
	decoder := yaml.NewDecoder(r)

	result := []*model.Pipeline{{}}

	err := decoder.Decode(result[0])
	if errors.Is(err, io.EOF) {
		err = nil
	}
	if err != nil {
		return nil, fmt.Errorf("error decoding pipeline: %w", err)
	}

	for jobName, job := range result[0].Jobs {
		job.Name = jobName
		if strings.Contains(jobName, ":") {
			job.Nested = true
		}
	}

	for taskName, task := range result[0].Tasks {
		task.Name = taskName
		if strings.Contains(taskName, ":") {
			task.Nested = true
		}
	}

	return result, nil
}
