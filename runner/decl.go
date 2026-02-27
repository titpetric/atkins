package runner

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/model"
)

// ProcessDecl processes an Decl and returns a map of variables.
// It handles:
// - Manual vars with interpolation ($(...), ${{ ... }})
// - Include files (.yml format)
// Vars take precedence over included files.
func ProcessDecl(ctx *ExecutionContext, decl *model.Decl) (map[string]any, error) {
	result := make(map[string]any)

	// First, load included files
	if decl != nil && decl.Include != nil {
		for _, filename := range decl.Include.Files {
			if err := loadYaml(filename, &result); err != nil {
				return nil, fmt.Errorf("failed to load vars file %q: %w", filename, err)
			}
		}
	}

	// Then, process and interpolate vars (they override included values)
	if decl != nil && decl.Vars != nil {
		interpolated, err := interpolateVariables(ctx, decl.Vars)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate vars: %w", err)
		}
		for k, v := range interpolated {
			result[k] = v
		}
	}

	return result, nil
}

func loadYaml(filename string, dest any) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, dest)
}

// MergeVariables merges variables from Decl into the execution context.
func MergeVariables(ctx *ExecutionContext, decl *model.Decl) error {
	if decl == nil {
		return nil
	}

	processed, err := ProcessDecl(ctx, decl)
	if err != nil {
		return fmt.Errorf("error processing variables: %w", err)
	}

	// Merge variables into context FIRST so they're available for env interpolation
	for k, v := range processed {
		ctx.Variables[k] = v
	}

	if decl.Env != nil {
		if err := mergeEnv(ctx, decl.Env); err != nil {
			return fmt.Errorf("error processing environment: %w", err)
		}
	}

	return nil
}
