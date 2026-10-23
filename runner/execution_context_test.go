package runner

import (
	"sync"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// resolveContext is a skill pipeline running among the pipelines of a
// project, each declaring a build job.
func resolveContext() (ctx *ExecutionContext, main, skill *model.Pipeline) {
	main = &model.Pipeline{Name: "main", Jobs: map[string]*model.Job{"build": {Name: "build"}}}
	skill = &model.Pipeline{ID: "go", Name: "go", Jobs: map[string]*model.Job{"build": {Name: "build"}}}
	return &ExecutionContext{Pipeline: skill, AllPipelines: []*model.Pipeline{main, skill}}, main, skill
}

func TestExecutionContext_Resolver(t *testing.T) {
	t.Parallel()

	ctx, _, skill := resolveContext()
	resolved, err := ctx.Resolver().Resolve("go:build")
	assert.NoError(t, err)
	assert.Equal(t, skill, resolved.Pipeline)
}

func TestExecutionContext_SkillResolver(t *testing.T) {
	t.Parallel()

	ctx, _, skill := resolveContext()
	resolved, err := ctx.SkillResolver().Resolve("build")
	assert.NoError(t, err)
	assert.Equal(t, skill, resolved.Pipeline)
}

func TestExecutionContext_Resolve(t *testing.T) {
	t.Parallel()

	ctx, main, skill := resolveContext()

	resolved, err := ctx.Resolve("build")
	assert.NoError(t, err)
	assert.Equal(t, skill, resolved.Pipeline, "a bare name resolves in the skill first")

	resolved, err = ctx.Resolve(":build")
	assert.NoError(t, err)
	assert.Equal(t, main, resolved.Pipeline, "a leading colon resolves globally")
}

func TestExecutionContext_Copy(t *testing.T) {
	t.Parallel()

	ctx := &ExecutionContext{
		Variables: NewContextVariables(map[string]any{"name": "a"}),
		Env:       Env{"HOME": "/a"},
		Depth:     1,
	}
	copied := ctx.Copy()
	copied.Variables.Set("name", "b")
	copied.Env["HOME"] = "/b"

	assert.Equal(t, "a", ctx.Variables.Get("name"))
	assert.Equal(t, "/a", ctx.Env["HOME"])
	assert.Equal(t, 2, copied.Depth)
}

func TestExecutionContext_MarkJobCompleted(t *testing.T) {
	t.Parallel()

	ctx := &ExecutionContext{jobTracker: newJobTracker()}
	copied := ctx.Copy()
	copied.MarkJobCompleted("build")

	assert.True(t, ctx.IsJobCompleted("build"), "copies share the tracker")
	assert.False(t, ctx.IsJobCompleted("test"))
}

func TestExecutionContext_IsJobCompleted(t *testing.T) {
	t.Parallel()

	ctx := &ExecutionContext{}
	ctx.MarkJobCompleted("build")
	assert.False(t, ctx.IsJobCompleted("build"), "a context without a tracker records nothing")
}

func TestExecutionContext_Render(t *testing.T) {
	t.Parallel()

	builder := treeview.NewBuilder("render")
	ctx := &ExecutionContext{Display: treeview.NewSilentDisplay(), Builder: builder}
	ctx.Render()
	assert.Equal(t, "render", builder.Root().GetName())
}

func TestExecutionContext_EmitProgress(t *testing.T) {
	t.Parallel()

	var got []JobProgressEvent
	ctx := &ExecutionContext{Progress: ProgressObserverFunc(func(ev JobProgressEvent) {
		got = append(got, ev)
	})}
	ctx.EmitProgress(JobProgressEvent{JobName: "build", Status: JobProgressRunning})
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "build", got[0].JobName)

	(&ExecutionContext{}).EmitProgress(JobProgressEvent{JobName: "dropped"})
}

func TestExecutionContext_NextStepIndex(t *testing.T) {
	t.Parallel()

	ctx := &ExecutionContext{}
	var (
		mu   sync.Mutex
		seen = map[int]bool{}
		wg   sync.WaitGroup
	)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			idx := ctx.NextStepIndex()
			mu.Lock()
			seen[idx] = true
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Equal(t, 20, len(seen), "every call takes its own index")
	assert.Equal(t, 20, ctx.NextStepIndex())
}
