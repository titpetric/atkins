package runner

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/titpetric/atkins/model"
)

// Env is a map of environment variables.
type Env map[string]string

// Environ returns the environment as a slice of KEY=VALUE strings.
func (e Env) Environ() []string {
	if e == nil {
		return nil
	}
	s := make([]string, 0, len(e))
	for k, v := range e {
		s = append(s, k+"="+v)
	}
	return s
}

// mergeEnv merges environment variables from EnvDecl into the execution context.
// Handles both workflow-level, job-level, and step-level env declarations.
func mergeEnv(ctx *ExecutionContext, decl *model.EnvDecl) error {
	if decl == nil {
		return nil
	}

	processed, err := processEnv(ctx, decl)
	if err != nil {
		return err
	}

	// Merge into context
	for k, v := range processed {
		ctx.Env[k] = v
	}

	return nil
}

// processEnv processes an EnvDecl and returns a map of environment variables.
// It handles:
// - Manual vars with interpolation ($(...), ${{ ... }})
// - Include files (.env format)
// Vars take precedence over included files.
func processEnv(ctx *ExecutionContext, decl *model.EnvDecl) (map[string]string, error) {
	result := make(map[string]string)

	// First, load included files
	if decl != nil && decl.Include != nil {
		for _, filePath := range decl.Include.Files {
			if err := loadEnvFile(filePath, result); err != nil {
				return nil, fmt.Errorf("failed to load env file %q: %w", filePath, err)
			}
		}
	}

	// Then, process and interpolate vars (they override included values)
	if decl != nil && decl.Vars != nil {
		interpolated, err := interpolateVariables(ctx, decl.Vars)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate env vars: %w", err)
		}
		for k, v := range interpolated {
			// Convert to string
			result[k] = fmt.Sprintf("%v", v)
		}
	}

	return result, nil
}

// loadEnvFile reads a .env file and populates the env map.
// Format: KEY=VALUE (one per line, # for comments)
func loadEnvFile(filePath string, env map[string]string) error {
	// Interpolate the file path in case it contains variables
	// For now, support simple shell expansion
	expandedPath := os.ExpandEnv(filePath)

	file, err := os.Open(expandedPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanned := 0
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		idx := strings.Index(line, "=")
		if idx == -1 {
			// Skip lines without =
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Handle quoted values
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}

		env[key] = value
		scanned++
	}

	if scanned == 0 {
		return fmt.Errorf("no envs found in %s", filePath)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
