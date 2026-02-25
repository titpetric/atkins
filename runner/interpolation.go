package runner

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/psexec"
)

// Matches ${{ variable_name }}
var interpolationRegex = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)

// InterpolateString replaces ${{ expression }} with values from context.
// Supports variable interpolation, dot notation, and expr expressions with ?? and || operators.
func InterpolateString(s string, ctx *ExecutionContext) (string, error) {
	result := s

	// Handle command execution: $(command)
	// Use manual parsing to handle nested parentheses correctly
	var cmdErr error
	result = extractAndProcessCommandSubstitutions(ctx, result, &cmdErr)

	if cmdErr != nil {
		return "", cmdErr
	}

	// Handle variable interpolation: ${{ expression }}
	result = interpolationRegex.ReplaceAllStringFunc(result, func(match string) string {
		exprStr := interpolationRegex.FindStringSubmatch(match)[1]
		exprStr = strings.TrimSpace(exprStr)

		// Evaluate expression using expr-lang
		val, err := evaluateExpression(exprStr, ctx)
		if err != nil {
			// If expression evaluation fails, return original match
			return match
		}

		// Convert result to string
		if val != nil {
			return fmt.Sprintf("%v", val)
		}

		// Return original if result is nil
		return match
	})

	return result, nil
}

// extractAndProcessCommandSubstitutions handles $(...) by properly matching nested parentheses
func extractAndProcessCommandSubstitutions(ctx *ExecutionContext, s string, cmdErr *error) string {
	if *cmdErr != nil {
		return s
	}

	result := ""
	i := 0
	for i < len(s) {
		// Look for $(
		if i < len(s)-1 && s[i] == '$' && s[i+1] == '(' {
			// Found start of command substitution
			// Find the matching closing paren
			closeIdx := findMatchingParen(s, i+2)
			if closeIdx == -1 {
				// No matching paren, treat as literal
				result += string(s[i])
				i++
				continue
			}

			// Extract the command (without the $( and ))
			cmd := s[i+2 : closeIdx]
			cmd = strings.TrimSpace(cmd)

			// First interpolate ${{ }} inside the command before executing it
			interpolatedCmd, err := interpolateVariablesInString(cmd, ctx)
			if err != nil {
				// Log the error but continue with original command
				interpolatedCmd = cmd
			}

			// Execute with context env variables
			startTime := time.Now()
			var startOffset float64
			if ctx.EventLogger != nil {
				startOffset = ctx.EventLogger.GetElapsed()
			}

			exec := psexec.NewWithOptions(&psexec.Options{
				DefaultDir: ctx.Dir,
				DefaultEnv: ctx.Env.Environ(),
			})
			cmdResult := exec.Run(context.Background(), exec.ShellCommand(interpolatedCmd))
			durationMs := time.Since(startTime).Milliseconds()

			// Log the command execution
			if ctx.EventLogger != nil {
				var parentID string
				if ctx.CurrentStep != nil {
					parentID = ctx.CurrentStep.ID
				} else if ctx.Job != nil {
					parentID = ctx.Job.Name
				}
				exitCode := cmdResult.ExitCode()
				errMsg := ""
				if !cmdResult.Success() {
					if cmdResult.Err() != nil {
						errMsg = cmdResult.Err().Error()
					}
				}
				ctx.EventLogger.LogCommand(eventlog.LogEntry{
					Type:       eventlog.EventTypeSubstitution,
					ID:         fmt.Sprintf("subst-%d", startTime.UnixNano()),
					ParentID:   parentID,
					Command:    interpolatedCmd,
					Dir:        ctx.Dir,
					Output:     strings.TrimSpace(cmdResult.Output()),
					Error:      errMsg,
					ExitCode:   exitCode,
					Start:      startOffset,
					DurationMs: durationMs,
				})
			}

			if !cmdResult.Success() {
				// Capture error with better context showing what command was executed
				errMsg := ""
				if cmdResult.Err() != nil {
					errMsg = cmdResult.Err().Error()
				}
				*cmdErr = fmt.Errorf("command execution failed in $(%s): %s", interpolatedCmd, errMsg)
				return s
			}
			result += strings.TrimSpace(cmdResult.Output())
			i = closeIdx + 1
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

// findMatchingParen finds the index of the closing parenthesis that matches the opening at startIdx
// startIdx should point to the character after the opening (
func findMatchingParen(s string, startIdx int) int {
	depth := 1
	i := startIdx

	for i < len(s) && depth > 0 {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '"', '\'':
			// Skip quoted strings to avoid counting parens inside quotes
			quote := s[i]
			i++
			for i < len(s) {
				if s[i] == quote {
					// Check if it's escaped
					if i > 0 && s[i-1] != '\\' {
						break
					}
					// Handle escaped backslash (\\")
					if i > 1 && s[i-2] == '\\' && s[i-1] == '\\' {
						break
					}
				}
				i++
			}
		}
		i++
	}

	if depth == 0 {
		return i - 1
	}
	return -1
}

// interpolateVariablesInString handles ${{ }} substitution within command strings
func interpolateVariablesInString(s string, ctx *ExecutionContext) (string, error) {
	result := s

	// Handle variable interpolation: ${{ expression }}
	result = interpolationRegex.ReplaceAllStringFunc(result, func(match string) string {
		exprStr := interpolationRegex.FindStringSubmatch(match)[1]
		exprStr = strings.TrimSpace(exprStr)

		// Evaluate expression using expr-lang
		val, err := evaluateExpression(exprStr, ctx)
		if err != nil {
			// If expression evaluation fails, log it but return original
			return match
		}

		// Convert result to string
		if val != nil {
			return fmt.Sprintf("%v", val)
		}

		// Return original if result is nil
		return match
	})

	return result, nil
}

// InterpolateMap recursively interpolates all string values in a map.
func InterpolateMap(ctx *ExecutionContext, m map[string]any) error {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			interpolated, err := InterpolateString(val, ctx)
			if err != nil {
				return err
			}
			m[k] = interpolated
		case map[string]any:
			if err := InterpolateMap(ctx, val); err != nil {
				return err
			}
		case []any:
			for i, item := range val {
				if str, ok := item.(string); ok {
					interpolated, err := InterpolateString(str, ctx)
					if err != nil {
						return err
					}
					val[i] = interpolated
				}
			}
		}
	}
	return nil
}

// InterpolateCommand interpolates a command string.
func InterpolateCommand(cmd string, ctx *ExecutionContext) (string, error) {
	return InterpolateString(cmd, ctx)
}

// evaluateExpression evaluates an expr expression with access to variables and environment.
// Uses expr-lang/expr for evaluation with support for:
//   - Simple variable access: varName
//   - Dot notation: user.name
//   - Null coalescing (RECOMMENDED): var ?? default
//   - Returns second value only if first is nil/missing
//   - Empty strings, false, 0 are valid and won't trigger default
//   - Complex expressions: (var1 ?? var2) ?? 'fallback'
//
// Note: The ?? (null coalescing) operator is the preferred pattern for defaults
// since it explicitly handles nil/missing values without side effects on falsy values.
func evaluateExpression(exprStr string, ctx *ExecutionContext) (any, error) {
	// Merge variables and environment into a single map for expr evaluation
	env := make(map[string]any)
	for k, v := range ctx.Variables {
		env[k] = v
	}
	for k, v := range ctx.Env {
		env[k] = v
	}

	// Compile and evaluate the expression
	program, err := expr.Compile(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return result, nil
}
