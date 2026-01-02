package runner

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/titpetric/atkins-ci/model"
	"github.com/titpetric/atkins-ci/spinner"
)

// TreeRenderer manages in-place tree rendering with ANSI cursor control
type TreeRenderer struct {
	lastLineCount int
	mu            sync.Mutex
}

// NewTreeRenderer creates a new tree renderer
func NewTreeRenderer() *TreeRenderer {
	return &TreeRenderer{
		lastLineCount: 0,
	}
}

// Render outputs the tree, updating in-place if previously rendered
func (tr *TreeRenderer) Render(tree *ExecutionTree) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	output := tree.RenderTree()
	lineCount := countOutputLines(output)

	if tr.lastLineCount > 0 {
		// Move cursor up, clear to end of display
		fmt.Printf("\033[%dA\033[J", tr.lastLineCount)
	}

	fmt.Print(output)
	tr.lastLineCount = lineCount
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

// Executor runs pipeline jobs and steps
type Executor struct{}

// NewExecutor creates a new executor
func NewExecutor() *Executor {
	return &Executor{}
}

// ExecuteJob runs a single job
func (e *Executor) ExecuteJob(ctx *model.ExecutionContext, jobName string, job *model.Job) error {
	// Merge job variables into context
	if job.Vars != nil {
		for k, v := range job.Vars {
			ctx.Variables[k] = v
		}
	}

	// Merge job environment
	if job.Env != nil {
		for k, v := range job.Env {
			ctx.Env[k] = v
		}
	}

	// Execute steps
	if len(job.Steps) > 0 {
		return e.executeSteps(ctx, job.Steps)
	}

	// Execute legacy cmd/cmds format
	if job.Run != "" {
		return e.executeCommand(ctx, job.Run)
	}

	if job.Cmd != "" {
		return e.executeCommand(ctx, job.Cmd)
	}

	if len(job.Cmds) > 0 {
		for _, cmd := range job.Cmds {
			if err := e.executeCommand(ctx, cmd); err != nil {
				return err
			}
		}
		return nil
	}

	return nil
}

// executeSteps runs a sequence of steps
func (e *Executor) executeSteps(ctx *model.ExecutionContext, steps []model.Step) error {
	// Pre-populate tree with all pending steps
	if jobNode, ok := ctx.CurrentJob.(*TreeNode); ok {
		for _, step := range steps {
			pendingNode := &TreeNode{
				Name:     step.Name,
				Status:   StatusPending,
				Children: make([]*TreeNode, 0),
			}
			jobNode.Children = append(jobNode.Children, pendingNode)
		}
		// Re-render tree to show all pending steps
		if tree, ok := ctx.Tree.(*ExecutionTree); ok {
			if renderer, ok := ctx.Renderer.(*TreeRenderer); ok {
				renderer.Render(tree)
			}
		}
	}

	for i, step := range steps {
		// Handle step-level environment variables
		stepCtx := &model.ExecutionContext{
			Variables: ctx.Variables,
			Env:       make(map[string]string),
			Results:   ctx.Results,
			QuietMode: ctx.QuietMode,
			Pipeline:  ctx.Pipeline,
			Job:       ctx.Job,
			Step:      step.Name,
			Depth:     ctx.Depth + 1,
		}

		// Copy parent env and add step-specific env
		for k, v := range ctx.Env {
			stepCtx.Env[k] = v
		}
		if step.Env != nil {
			for k, v := range step.Env {
				stepCtx.Env[k] = v
			}
		}

		// Determine which command to run
		var cmd string
		if step.Run != "" {
			cmd = step.Run
		} else if step.Cmd != "" {
			cmd = step.Cmd
		} else if len(step.Cmds) > 0 {
			cmd = strings.Join(step.Cmds, " && ")
		} else {
			continue
		}

		// Get existing step node from tree (already added as pending)
		var stepNode *TreeNode
		if jobNode, ok := ctx.CurrentJob.(*TreeNode); ok {
			if i < len(jobNode.Children) {
				stepNode = jobNode.Children[i]
			}
		}

		// Mark step as running
		if stepNode != nil {
			stepNode.SetStatus(StatusRunning)
		}

		// Start spinner and execute command
		s := spinner.New()
		s.Start()

		// Channel to signal command completion
		cmdDone := make(chan error)
		go func() {
			cmdDone <- e.executeCommand(stepCtx, cmd)
		}()

		// Update spinner in tree while command runs
		tickerTicker := time.NewTicker(100 * time.Millisecond)
		defer tickerTicker.Stop()

		for {
			select {
			case err := <-cmdDone:
				s.Stop()
				tickerTicker.Stop()

				// Update tree node status
				if stepNode != nil {
					if err != nil {
						stepNode.SetStatus(StatusFailed)
						return err
					}
					stepNode.SetStatus(StatusPassed)
					// Clear error log on successful step
					ErrorLogMutex.Lock()
					ErrorLog.Reset()
					ErrorLogMutex.Unlock()
				}

				// Increment step counters
				ctx.StepsCount++
				ctx.StepsPassed++

				// Render tree with final state
				if tree, ok := ctx.Tree.(*ExecutionTree); ok {
					if renderer, ok := ctx.Renderer.(*TreeRenderer); ok {
						renderer.Render(tree)
					}
				}
				goto nextStep

			case <-tickerTicker.C:
				if stepNode != nil {
					stepNode.SetSpinner(s.String())
					// Render tree with updated spinner
					if tree, ok := ctx.Tree.(*ExecutionTree); ok {
						if renderer, ok := ctx.Renderer.(*TreeRenderer); ok {
							renderer.Render(tree)
						}
					}
				}
			}
		}

	nextStep:
	}
	return nil
}

// countOutputLines counts the number of newlines in output
func countOutputLines(output string) int {
	count := 0
	for _, ch := range output {
		if ch == '\n' {
			count++
		}
	}
	return count
}

// executeCommand runs a single command with interpolation
func (e *Executor) executeCommand(ctx *model.ExecutionContext, cmd string) error {
	// Interpolate the command
	interpolated, err := InterpolateCommand(cmd, ctx)
	if err != nil {
		return fmt.Errorf("interpolation failed: %w", err)
	}

	// Execute the command via bash with quiet mode
	output, err := ExecuteCommandWithQuiet(interpolated, ctx.QuietMode)
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	// Only print output if not in quiet mode (quiet mode 1 = suppress output)
	if ctx.QuietMode == 0 && output != "" {
		fmt.Print(output)
	}

	return nil
}
