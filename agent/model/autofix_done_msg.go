package model

import (
	"time"

	pipeline "github.com/titpetric/atkins/model"
)

// AutofixDoneMsg signals an autofix completed.
type AutofixDoneMsg struct {
	OriginalTask *pipeline.ResolvedTask
	Err          error
	Duration     time.Duration
}
