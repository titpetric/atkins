package runner

import (
	"github.com/titpetric/atkins/model"
)

// IterationContext holds the variables for a single iteration of a for loop.
//
// ExpandFor returns one per iteration. An iterator reads "item in items",
// where items names a variable, "(index, item) in items" or
// "(key, value) in items", and any of them can take its items from a shell
// command: "item in $(ls ./bin/*.test)".
type IterationContext struct {
	Variables model.VariableStorage
}
