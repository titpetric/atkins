package model

import pipeline "github.com/titpetric/atkins/model"

// AutofixStartMsg signals an autofix should begin.
type AutofixStartMsg struct {
	OriginalTask *pipeline.ResolvedTask
	FixTask      *pipeline.ResolvedTask
}
