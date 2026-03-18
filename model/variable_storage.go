package model

// VariableStorage provides access to variables with lazy evaluation support.
// Implementations may defer evaluation until Get is called.
type VariableStorage interface {
	// Get returns a variable value. If the variable is pending evaluation,
	// it will be evaluated on first access and cached.
	Get(key string) any

	// Set stores a value directly (already evaluated).
	// Used for loop variables and runtime-computed values.
	Set(key string, value any)

	// Clone creates an independent copy for iteration scopes.
	Clone() VariableStorage

	// Walk iterates over all evaluated variables.
	// The function is called for each key-value pair while holding a lock.
	Walk(fn func(key string, value any))
}
