package runner

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/model"
)

// ProcessDecl processes a Decl and returns a map of variables.
// It handles:
// - Manual vars with interpolation ($(...), ${{ ... }})
// - Include files (.yml format)
// Vars take precedence over included files.
func ProcessDecl(ctx *ExecutionContext, decl *model.Decl) (map[string]any, error) {
	result := make(map[string]any)

	// First, load included files
	if decl != nil && decl.Include != nil {
		for _, filename := range decl.Include.Files {
			if err := loadVarsFile(filename, result); err != nil {
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

// loadVarsFile reads a YAML file and merges its contents into the vars map.
func loadVarsFile(filename string, vars map[string]any) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &vars)
}

// MergeSkillVariables merges variables from a skill's Decl into the context.
// When depth > 1, variables are already on the stack from a parent pipeline,
// so new vars are treated as defaults (existing keys are preserved).
// At depth <= 1 it behaves identically to MergeVariables.
func MergeSkillVariables(ctx *ExecutionContext, decl *model.Decl) error {
	if decl == nil {
		return nil
	}
	if ctx.Depth <= 1 {
		return MergeVariables(ctx, decl)
	}
	// Snapshot existing variable values
	existing := make(map[string]any)
	if ctx.Variables != nil {
		ctx.Variables.Walk(func(k string, v any) {
			existing[k] = v
		})
	}
	// Merge normally (may overwrite)
	if err := MergeVariables(ctx, decl); err != nil {
		return err
	}
	// Restore: existing values take precedence over newly merged ones
	for k, v := range existing {
		ctx.Variables.Set(k, v)
	}
	return nil
}

// MergeVariables merges variables from Decl into the execution context.
// When both vars and env.vars are present, they are resolved together using
// a unified dependency graph so that cross-references work correctly
// (e.g., vars using $(echo $ENV_VAR) and env using ${{ var_name }}).
func MergeVariables(ctx *ExecutionContext, decl *model.Decl) error {
	if decl == nil {
		return nil
	}

	hasVars := decl.Vars != nil && len(decl.Vars) > 0
	hasEnvVars := decl.Env != nil && decl.Env.Vars != nil && len(decl.Env.Vars) > 0

	// When both vars and env.vars have entries, use unified resolution
	// to handle cross-dependencies correctly.
	if hasVars && hasEnvVars {
		r, err := newResolver(ctx, decl)
		if err != nil {
			return err
		}
		return r.mergeInto(ctx)
	}

	// Otherwise, use the original sequential path.
	processed, err := ProcessDecl(ctx, decl)
	if err != nil {
		return fmt.Errorf("error processing variables: %w", err)
	}

	for k, v := range processed {
		ctx.Variables.Set(k, v)
	}

	if decl.Env != nil {
		if err := mergeEnv(ctx, decl.Env); err != nil {
			return fmt.Errorf("error processing environment: %w", err)
		}
	}

	return nil
}
