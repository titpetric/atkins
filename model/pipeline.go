package model

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
)

// Pipeline represents the root structure of an atkins.yml file
type Pipeline struct {
	Name  string          `yaml:"name,omitempty"`
	Jobs  map[string]*Job `yaml:"jobs,omitempty"`
	Tasks map[string]*Job `yaml:"tasks,omitempty"`
}

// Job represents a job/task in the pipeline
type Job struct {
	Desc      string                 `yaml:"desc,omitempty"`
	RunsOn    string                 `yaml:"runs_on,omitempty"`
	Container string                 `yaml:"container,omitempty"`
	If        string                 `yaml:"if,omitempty"`
	Cmd       string                 `yaml:"cmd,omitempty"`
	Cmds      []string               `yaml:"cmds,omitempty"`
	Run       string                 `yaml:"run,omitempty"`
	Steps     []Step                 `yaml:"steps,omitempty"`
	Services  map[string]*Service    `yaml:"services,omitempty"`
	Vars      map[string]interface{} `yaml:"vars,omitempty"`
	Env       map[string]string      `yaml:"env,omitempty"`
	Detach    bool                   `yaml:"detach,omitempty"`
	DependsOn interface{}            `yaml:"depends_on,omitempty"` // string or []string
	Timeout   string                 `yaml:"timeout,omitempty"`    // e.g., "10m", "300s"

	Name   string `yaml:"-"`
	Nested bool   `yaml:"-"`
}

// Step represents a step within a job
type Step struct {
	Name     string                 `yaml:"name,omitempty"`
	Desc     string                 `yaml:"desc,omitempty"`
	Run      string                 `yaml:"run,omitempty"`
	Cmd      string                 `yaml:"cmd,omitempty"`
	Cmds     []string               `yaml:"cmds,omitempty"`
	If       string                 `yaml:"if,omitempty"`
	For      string                 `yaml:"for,omitempty"`
	Env      map[string]string      `yaml:"env,omitempty"`
	Uses     string                 `yaml:"uses,omitempty"`
	With     map[string]interface{} `yaml:"with,omitempty"`
	Detach   bool                   `yaml:"detach,omitempty"`
	Defer    string                 `yaml:"defer,omitempty"`
	Deferred bool                   `yaml:"deferred,omitempty"`

	// Internal: cached compiled expr program for If evaluation
	ifProgram *vm.Program
}

// UnmarshalYAML implements custom unmarshalling for Step to support various formats
func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		// Simple string step - treat as a Run command
		s.Run = node.Value
		s.Name = node.Value
		return nil
	}

	if node.Kind == yaml.MappingNode {
		// Object step - use default unmarshalling
		type rawStep Step
		if err := node.Decode((*rawStep)(s)); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("invalid step format: expected string or object, got %v", node.Kind)
}

// IterationContext holds the variables for a single iteration of a for loop
type IterationContext struct {
	Variables map[string]interface{}
}

// EvaluateIf evaluates the If condition using expr-lang
// Returns true if the condition is met, false if no condition or condition is false
// Returns error only for invalid expressions
func (s *Step) EvaluateIf(ctx *ExecutionContext) (bool, error) {
	if s.If == "" {
		return true, nil // No condition means always execute
	}

	// Compile and cache the expression program
	if s.ifProgram == nil {
		prog, err := expr.Compile(s.If, expr.AllowUndefinedVariables())
		if err != nil {
			return false, fmt.Errorf("failed to compile if expression %q: %w", s.If, err)
		}
		s.ifProgram = prog
	}

	// Build the environment for expression evaluation
	env := make(map[string]interface{})

	// Add all context variables
	for k, v := range ctx.Variables {
		env[k] = v
	}

	// Add environment variables
	for k, v := range ctx.Env {
		env[k] = v
	}

	// Run the compiled program
	result, err := expr.Run(s.ifProgram, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate if expression %q: %w", s.If, err)
	}

	// Coerce the result to boolean
	switch v := result.(type) {
	case bool:
		return v, nil
	case nil:
		return false, nil
	case string:
		return v != "" && v != "false" && v != "0", nil
	case int, int32, int64:
		return result != 0, nil
	case float32, float64:
		return result != 0.0, nil
	default:
		return true, nil // Non-zero/non-nil values are truthy
	}
}

// ExpandFor expands a for loop into multiple iteration contexts
// Supports patterns:
//   - "item in items" (items is a variable name)
//   - "(index, item) in items"
//   - "(key, value) in items"
//   - Any of the above with bash expansion: "item in $(ls ./bin/*.test)"
func (s *Step) ExpandFor(ctx *ExecutionContext, executeCommand func(string) (string, error)) ([]IterationContext, error) {
	if s.For == "" {
		return nil, nil
	}

	// Parse the for loop pattern
	itemsVar, loopVar, indexVar, keyVar, err := parseForPattern(s.For)
	if err != nil {
		return nil, fmt.Errorf("invalid for loop syntax: %w", err)
	}

	// Get the items list
	items, err := getForItems(itemsVar, ctx, executeCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to get items for 'for: %s': %w", s.For, err)
	}

	if len(items) == 0 {
		return []IterationContext{}, nil
	}

	// Build iteration contexts based on the pattern
	var result []IterationContext

	if indexVar != "" || keyVar != "" {
		// (index, item) or (key, value) pattern
		for i, item := range items {
			vars := make(map[string]interface{})
			for k, v := range ctx.Variables {
				vars[k] = v
			}

			// Check if this is a map for (key, value) iteration
			if mapItem, ok := item.(map[string]interface{}); ok && indexVar != "" && keyVar != "" {
				// Could be either (index, item) with a map item, or (key, value) iteration
				// If items contains only one map, treat as (key, value)
				if len(items) == 1 {
					for k, v := range mapItem {
						vars[indexVar] = k // First var is the key
						vars[keyVar] = v   // Second var is the value
						// Process each key-value pair as a separate iteration
						result = append(result, IterationContext{Variables: copyMap(vars)})
					}
					continue
				}
			}

			if indexVar != "" && keyVar != "" {
				// (index, item) pattern
				vars[indexVar] = i
				vars[keyVar] = item
			} else if keyVar != "" {
				// Fallback for single var with key case
				vars[indexVar] = i
				vars[keyVar] = item
			}
			result = append(result, IterationContext{Variables: vars})
		}
	} else {
		// Simple "item in items" or "name in names" pattern
		// Use the actual loop variable name (loopVar)
		for _, item := range items {
			vars := make(map[string]interface{})
			for k, v := range ctx.Variables {
				vars[k] = v
			}
			vars[loopVar] = item
			result = append(result, IterationContext{Variables: vars})
		}
	}

	return result, nil
}

// parseForPattern parses for loop patterns and returns (itemsVar, loopVar, indexVar, keyVar, error)
// Patterns: "item in items", "(idx, item) in items", "(key, value) in items"
func parseForPattern(forSpec string) (string, string, string, string, error) {
	forSpec = strings.TrimSpace(forSpec)

	// Match "(var1, var2) in items" or "var in items"
	parenPattern := regexp.MustCompile(`^\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)\s+in\s+(.+)$`)
	simplePattern := regexp.MustCompile(`^\s*(\w+)\s+in\s+(.+)$`)

	if matches := parenPattern.FindStringSubmatch(forSpec); matches != nil {
		// (key, value) or (idx, item)
		// For 2-var pattern, return itemsVar, loopVar="", indexVar, keyVar
		return matches[3], "", matches[1], matches[2], nil
	}

	if matches := simplePattern.FindStringSubmatch(forSpec); matches != nil {
		// item in items
		// For simple pattern: loopVar is the variable name
		return matches[2], matches[1], "", "", nil
	}

	return "", "", "", "", fmt.Errorf("unrecognized for pattern, expected 'item in items' or '(idx, item) in items'")
}

// getForItems retrieves the items list for a for loop
// itemsSpec can be:
//   - A variable name: "items"
//   - A bash command: "$(ls ./bin/*.test)"
func getForItems(itemsSpec string, ctx *ExecutionContext, executeCommand func(string) (string, error)) ([]interface{}, error) {
	itemsSpec = strings.TrimSpace(itemsSpec)

	// Check for bash command expansion: $(...)
	if strings.HasPrefix(itemsSpec, "$(") && strings.HasSuffix(itemsSpec, ")") {
		cmd := itemsSpec[2 : len(itemsSpec)-1]
		output, err := executeCommand(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to execute command %q: %w", cmd, err)
		}

		// Split output by newlines
		lines := strings.Split(strings.TrimSpace(output), "\n")
		items := make([]interface{}, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				items = append(items, line)
			}
		}
		return items, nil
	}

	// Look up in variables
	if val, ok := ctx.Variables[itemsSpec]; ok {
		// Convert to []interface{}
		switch v := val.(type) {
		case []interface{}:
			return v, nil
		case []string:
			items := make([]interface{}, len(v))
			for i, s := range v {
				items[i] = s
			}
			return items, nil
		case string:
			// Single item
			return []interface{}{v}, nil
		case map[string]interface{}:
			// For key-value, return the map as a single item
			return []interface{}{v}, nil
		default:
			return []interface{}{v}, nil
		}
	}

	return nil, fmt.Errorf("variable %q not found in context", itemsSpec)
}

// copyMap creates a shallow copy of a map
func copyMap(m map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range m {
		copy[k] = v
	}
	return copy
}

// Service represents a service (e.g., Docker container) used in a job
type Service struct {
	Image    string            `yaml:"image,omitempty"`
	Pull     string            `yaml:"pull,omitempty"`
	Options  string            `yaml:"options,omitempty"`
	Ports    []string          `yaml:"ports,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Networks []string          `yaml:"networks,omitempty"`
}

// ExecutionContext holds runtime state during pipeline execution
type ExecutionContext struct {
	Variables   map[string]interface{}
	Env         map[string]string
	Results     map[string]interface{}
	QuietMode   int             // 0 = normal, 1 = quiet (no stdout), 2 = very quiet (no stdout/stderr)
	Pipeline    string          // Current pipeline name
	Job         string          // Current job name
	JobDesc     string          // Current job description
	Step        string          // Current step name
	Depth       int             // Nesting depth for indentation
	StepsCount  int             // Total number of steps executed
	StepsPassed int             // Number of steps that passed
	Tree        interface{}     // *ExecutionTree (avoid circular import)
	CurrentJob  interface{}     // *TreeNode for current job
	CurrentStep interface{}     // *TreeNode for current step
	Renderer    interface{}     // *TreeRenderer for in-place rendering
	Context     context.Context // Context for timeout and cancellation
}
