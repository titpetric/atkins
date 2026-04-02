package agent_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/agent"
)

func TestBreadcrumb_NewBreadcrumb(t *testing.T) {
	b := agent.NewBreadcrumb()
	assert.NotNil(t, b)
	assert.Empty(t, b.String())
}

func TestBreadcrumb_Push(t *testing.T) {
	b := agent.NewBreadcrumb()

	b.Push("go")
	assert.Equal(t, "go", b.String())

	b.Push("test")
	assert.Equal(t, "go > test", b.String())

	b.Push("step 1")
	assert.Equal(t, "go > test > step 1", b.String())
}

func TestBreadcrumb_Pop(t *testing.T) {
	b := agent.NewBreadcrumb()

	b.Push("go")
	b.Push("test")
	b.Push("step 1")

	b.Pop()
	assert.Equal(t, "go > test", b.String())

	b.Pop()
	assert.Equal(t, "go", b.String())

	b.Pop()
	assert.Empty(t, b.String())

	// Pop on empty should not panic
	b.Pop()
	assert.Empty(t, b.String())
}

func TestBreadcrumb_SetStatus(t *testing.T) {
	b := agent.NewBreadcrumb()

	b.Push("go")
	b.Push("test")
	b.SetStatus("running...")
	assert.Contains(t, b.String(), "[running...]")

	b.SetStatus("passed")
	assert.Contains(t, b.String(), "[passed")
}

func TestBreadcrumb_Clear(t *testing.T) {
	b := agent.NewBreadcrumb()

	b.Push("go")
	b.Push("test")
	b.SetStatus("done")

	b.Clear()
	assert.Empty(t, b.String())
}

func TestBreadcrumb_LastSegment(t *testing.T) {
	b := agent.NewBreadcrumb()

	assert.Empty(t, b.LastSegment())

	b.Push("go")
	assert.Equal(t, "go", b.LastSegment())

	b.Push("test")
	assert.Equal(t, "test", b.LastSegment())

	b.Pop()
	assert.Equal(t, "go", b.LastSegment())
}

func TestBreadcrumb_DurationDisplay(t *testing.T) {
	b := agent.NewBreadcrumb()

	b.Push("task")
	// Sleep briefly to get a non-zero duration
	time.Sleep(10 * time.Millisecond)

	b.SetStatus("done")
	str := b.String()

	// Should contain duration when status is "done"
	assert.Contains(t, str, "[done")
	// Duration should be displayed
	assert.True(t, strings.Contains(str, "ms") || strings.Contains(str, "s"),
		"expected duration in output: %s", str)
}
