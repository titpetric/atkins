package model

// VariableStorage stores variables with lazy evaluation support.
// Implementations may defer evaluation until the first Get call.
type VariableStorage interface {
	// Get returns a variable value. If the variable is pending evaluation,
	// Get evaluates it on first access and caches the result.
	Get(key string) any

	// Set stores a value directly (already evaluated).
	// Used for loop variables and runtime-computed values.
	Set(key string, value any)

	// Clone creates an independent copy for iteration scopes.
	Clone() VariableStorage

	// Walk iterates over all evaluated variables.
	// Walk calls the function for each key-value pair while holding a lock.
	Walk(fn func(key string, value any))
}
