package runner

import "github.com/titpetric/atkins/model"

// IterationContext holds the variables for a single iteration of a for loop.
type IterationContext struct {
	Variables model.VariableStorage
}
