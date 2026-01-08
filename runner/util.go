package runner

// copyVariables creates a shallow copy of a variables map
func copyVariables(vars map[string]any) map[string]any {
	copy := make(map[string]any)
	for k, v := range vars {
		copy[k] = v
	}
	return copy
}

// copyEnv creates a shallow copy of a variables map
func copyEnv(vars map[string]string) map[string]string {
	copy := make(map[string]string)
	for k, v := range vars {
		copy[k] = v
	}
	return copy
}
