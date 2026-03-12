package errors

import "github.com/titpetric/atkins/model"

// NoDefaultJobError is returned when no default job is found.
type NoDefaultJobError struct {
	Jobs map[string]*model.Job
}

// Error returns the error hinting a default job should be defined.
func (e *NoDefaultJobError) Error() string {
	return "task \"default\" does not exist"
}
