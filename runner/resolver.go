package runner

import (
	"fmt"
	"strings"

	"github.com/titpetric/atkins/model"
)

// Resolver resolves vars and env.vars together using a unified dependency
// graph. This handles cross-dependencies where vars use $(echo $ENV_VAR)
// and env uses ${{ var_name }}.
type Resolver struct {
	vars    map[string]any
	envVars map[string]any

	baseVars map[string]any
	baseEnv  map[string]string

	workCtx *ExecutionContext
}

// newResolver loads includes and seeds a working context for unified resolution.
func newResolver(ctx *ExecutionContext, decl *model.Decl) (*Resolver, error) {
	r := &Resolver{
		vars:    decl.Vars,
		envVars: decl.Env.Vars,
	}

	if err := r.loadIncludes(decl); err != nil {
		return nil, err
	}

	r.seedContext(ctx)
	return r, nil
}

// loadIncludes loads include files for both vars and env before resolution.
func (r *Resolver) loadIncludes(decl *model.Decl) error {
	r.baseVars = make(map[string]any)
	if decl.Include != nil {
		for _, filename := range decl.Include.Files {
			if err := loadVarsFile(filename, r.baseVars); err != nil {
				return fmt.Errorf("error processing variables: failed to load vars file %q: %w", filename, err)
			}
		}
	}

	r.baseEnv = make(map[string]string)
	if decl.Env != nil && decl.Env.Include != nil {
		for _, filePath := range decl.Env.Include.Files {
			if err := loadEnvFile(filePath, r.baseEnv); err != nil {
				return fmt.Errorf("error processing environment: failed to load env file %q: %w", filePath, err)
			}
		}
	}

	return nil
}

// seedContext creates the working ExecutionContext from existing state and base includes.
func (r *Resolver) seedContext(ctx *ExecutionContext) {
	workVars := ctx.Variables.Clone()
	for k, v := range r.baseVars {
		workVars.Set(k, v)
	}
	r.workCtx = &ExecutionContext{
		Variables:   workVars,
		Env:         make(map[string]string),
		Dir:         ctx.Dir,
		EventLogger: ctx.EventLogger,
	}
	for k, v := range ctx.Env {
		r.workCtx.Env[k] = v
	}
	for k, v := range r.baseEnv {
		r.workCtx.Env[k] = v
	}
}

// buildOrder builds a unified dependency graph across vars and env entries
// and returns the topologically sorted resolution order.
func (r *Resolver) buildOrder() ([]string, error) {
	deps := make(map[string][]string)
	for k, v := range r.vars {
		if strVal, ok := v.(string); ok {
			deps[nodePrefixVar+k] = extractUnifiedDependencies(strVal, r.vars, r.envVars)
		} else {
			deps[nodePrefixVar+k] = nil
		}
	}
	for k, v := range r.envVars {
		if strVal, ok := v.(string); ok {
			deps[nodePrefixEnv+k] = extractUnifiedDependencies(strVal, r.vars, r.envVars)
		} else {
			deps[nodePrefixEnv+k] = nil
		}
	}
	return topologicalSort(deps)
}

// resolve walks nodes in topological order, interpolating each value
// and accumulating results into the working context.
func (r *Resolver) resolve(order []string) (map[string]any, map[string]string, error) {
	resolvedVars := make(map[string]any)
	resolvedEnv := make(map[string]string)

	for _, nodeID := range order {
		switch {
		case strings.HasPrefix(nodeID, nodePrefixVar):
			k := strings.TrimPrefix(nodeID, nodePrefixVar)
			v, err := r.resolveValue(r.vars[k])
			if err != nil {
				return nil, nil, fmt.Errorf("error processing variables: failed to interpolate variable %q: %w", k, err)
			}
			resolvedVars[k] = v
			r.workCtx.Variables.Set(k, v)

		case strings.HasPrefix(nodeID, nodePrefixEnv):
			k := strings.TrimPrefix(nodeID, nodePrefixEnv)
			v, err := r.resolveValue(r.envVars[k])
			if err != nil {
				return nil, nil, fmt.Errorf("error processing environment: failed to interpolate env vars: failed to interpolate variable %q: %w", k, err)
			}
			s := fmt.Sprintf("%v", v)
			resolvedEnv[k] = s
			r.workCtx.Env[k] = s
		}
	}

	return resolvedVars, resolvedEnv, nil
}

// resolveValue interpolates a single value if it's a string, otherwise passes it through.
func (r *Resolver) resolveValue(v any) (any, error) {
	strVal, ok := v.(string)
	if !ok {
		return v, nil
	}
	return InterpolateString(strVal, r.workCtx)
}

// mergeInto runs the full resolution and writes results into ctx.
func (r *Resolver) mergeInto(ctx *ExecutionContext) error {
	order, err := r.buildOrder()
	if err != nil {
		return fmt.Errorf("error processing variables: %w", err)
	}

	resolvedVars, resolvedEnv, err := r.resolve(order)
	if err != nil {
		return err
	}

	for k, v := range r.baseVars {
		ctx.Variables.Set(k, v)
	}
	for k, v := range resolvedVars {
		ctx.Variables.Set(k, v)
	}
	for k, v := range r.baseEnv {
		ctx.Env[k] = v
	}
	for k, v := range resolvedEnv {
		ctx.Env[k] = v
	}

	return nil
}
