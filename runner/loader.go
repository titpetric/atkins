package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/titpetric/atkins-ci/model"
	"gopkg.in/yaml.v3"
)

// LoadPipeline loads and parses a pipeline from a yaml file
// Returns the number of documents loaded, the parsed pipeline, and any error
func LoadPipeline(filePath string) ([]*model.Pipeline, error) {
	// Read the raw file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	// Parse with plain YAML first (no expression evaluation)
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))

	var result = []*model.Pipeline{
		&model.Pipeline{},
	}
	if err := decoder.Decode(result[0]); err != nil {
		return nil, fmt.Errorf("error decoding pipeline: %w", err)
	}

	for jobName, job := range result[0].Jobs {
		job.Name = jobName
		if strings.Contains(jobName, ":") {
			job.Nested = true
		}
	}

	return result, nil
}
