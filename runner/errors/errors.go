package errors

import (
	"github.com/titpetric/atkins/model"
)

// NoDefaultJobError reports a pipeline that has no default job.
type NoDefaultJobError struct {
	Jobs map[string]*model.Job
}

// Error returns the error hinting that the pipeline needs a default job.
func (e *NoDefaultJobError) Error() string {
	return "task \"default\" does not exist"
}
