package runner

import (
	"fmt"
	"strings"
)

const (
	nodePrefixVar = "var:"
	nodePrefixEnv = "env:"
)

// extractVariableDependencies extracts variable names referenced via ${{ varName }} in a string.
// Only returns dependencies that exist in the vars map.
func extractVariableDependencies(s string, vars map[string]any) []string {
	matches := interpolationRegex.FindAllStringSubmatch(s, -1)
	var deps []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			varName := strings.TrimSpace(match[1])
			if _, exists := vars[varName]; exists && !seen[varName] {
				deps = append(deps, varName)
				seen[varName] = true
			}
		}
	}
	return deps
}

// extractUnifiedDependencies extracts prefixed dependency node IDs from a string value.
// It detects:
//   - ${{ name }} references → depends on var:name or env:name
//   - $NAME / ${NAME} inside $(...) blocks → depends on env:NAME
func extractUnifiedDependencies(s string, vars map[string]any, envVars map[string]any) []string {
	seen := make(map[string]bool)
	var deps []string

	addDep := func(id string) {
		if !seen[id] {
			seen[id] = true
			deps = append(deps, id)
		}
	}

	// 1. Find ${{ name }} references — prefer var, fall back to env
	matches := interpolationRegex.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			if _, ok := vars[name]; ok {
				addDep(nodePrefixVar + name)
			} else if _, ok := envVars[name]; ok {
				addDep(nodePrefixEnv + name)
			}
		}
	}

	// 2. Find $NAME / ${NAME} inside $(...) blocks — depends on env entries
	cmdBodies := extractCommandSubstitutionBodies(s)
	for _, body := range cmdBodies {
		for _, ref := range extractShellVarRefs(body) {
			if _, ok := envVars[ref]; ok {
				addDep(nodePrefixEnv + ref)
			}
		}
	}

	return deps
}

// extractCommandSubstitutionBodies returns the bodies of all $(...) blocks in s
// without executing them. Uses the same nesting-aware parser as interpolation.
func extractCommandSubstitutionBodies(s string) []string {
	var bodies []string
	i := 0
	for i < len(s) {
		if i < len(s)-1 && s[i] == '$' && s[i+1] == '(' {
			closeIdx := findMatchingParen(s, i+2)
			if closeIdx == -1 {
				i++
				continue
			}
			bodies = append(bodies, s[i+2:closeIdx])
			i = closeIdx + 1
		} else {
			i++
		}
	}
	return bodies
}

// extractShellVarRefs extracts shell variable references ($NAME and ${NAME})
// from a command string. Skips $( (command substitution) and ${{ (expr syntax).
func extractShellVarRefs(cmd string) []string {
	seen := make(map[string]bool)
	var refs []string
	i := 0
	for i < len(cmd) {
		if cmd[i] != '$' {
			i++
			continue
		}
		i++ // skip $
		if i >= len(cmd) {
			break
		}
		switch {
		case cmd[i] == '(':
			// $( — command substitution, skip
			i++
		case cmd[i] == '{':
			if i+1 < len(cmd) && cmd[i+1] == '{' {
				// ${{ — expr syntax, skip
				i += 2
			} else {
				// ${NAME} form
				i++ // skip {
				start := i
				for i < len(cmd) && isShellNameChar(cmd[i]) {
					i++
				}
				if i < len(cmd) && cmd[i] == '}' && i > start {
					name := cmd[start:i]
					if !seen[name] {
						seen[name] = true
						refs = append(refs, name)
					}
					i++ // skip }
				}
			}
		default:
			// $NAME form
			if isShellNameStart(cmd[i]) {
				start := i
				for i < len(cmd) && isShellNameChar(cmd[i]) {
					i++
				}
				name := cmd[start:i]
				if !seen[name] {
					seen[name] = true
					refs = append(refs, name)
				}
			}
		}
	}
	return refs
}

func isShellNameStart(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

func isShellNameChar(c byte) bool {
	return isShellNameStart(c) || (c >= '0' && c <= '9')
}

// topologicalSort performs a topological sort on the dependency graph.
// Returns an error if a cycle is detected.
func topologicalSort(deps map[string][]string) ([]string, error) {
	visited := make(map[string]int) // 0=unvisited, 1=visiting, 2=visited
	var order []string

	var visit func(node string) error
	visit = func(node string) error {
		if visited[node] == 1 {
			return fmt.Errorf("cycle detected involving variable %q", node)
		}
		if visited[node] == 2 {
			return nil
		}
		visited[node] = 1
		for _, dep := range deps[node] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visited[node] = 2
		order = append(order, node)
		return nil
	}

	for node := range deps {
		if visited[node] == 0 {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}
	return order, nil
}
