package model_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestPipelineWhen_IsEmpty checks a block without files is empty, a nil
// block included.
func TestPipelineWhen_IsEmpty(t *testing.T) {
	t.Parallel()

	var none *model.PipelineWhen
	assert.True(t, none.IsEmpty())
	assert.True(t, (&model.PipelineWhen{Here: true}).IsEmpty())
	assert.False(t, (&model.PipelineWhen{Files: []string{"compose.yml"}}).IsEmpty())
}
