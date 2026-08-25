// Package ai is the last-resort LLM fallback for the atkins agent. When
// local routing can't resolve an input to a task, an alias or a shell
// command, the input and the available pipelines are handed to `claude -p`,
// and the JSON reply becomes either atkins commands to run or a short
// message to print.
package ai

import (
	"context"
	"errors"
	"os/exec"

	"github.com/titpetric/atkins/model"
)

// ErrEmptyResponse is returned when claude replies with neither commands
// nor a message.
var ErrEmptyResponse = errors.New("ai returned an empty response")

// Result is what claude answered, after validation: atkins commands to run,
// or a message describing what it did instead. Exactly one of the two is
// set.
type Result struct {
	// Cmds holds one argv per command. Each has been checked to invoke
	// atkins itself, see ValidateAtkinsCmds.
	Cmds [][]string

	// Message is a short description of a change claude made, for inputs
	// that asked for an edit rather than for a job to run.
	Message string
}

// Available reports whether the claude CLI is on PATH. Both agent entry
// points call this to decide whether the fallback exists at all.
func Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// Ask sends an input local routing couldn't resolve to claude and returns
// the validated answer. Commands are not run here; the caller decides
// whether to confirm them first.
func Ask(ctx context.Context, input string, pipelines []*model.Pipeline, workDir string) (*Result, error) {
	resp, err := Invoke(ctx, BuildPrompt(input, pipelines, workDir))
	if err != nil {
		return nil, err
	}

	if len(resp.Cmds) > 0 {
		argvs, err := ValidateAtkinsCmds(resp.Cmds)
		if err != nil {
			return nil, err
		}
		return &Result{Cmds: argvs}, nil
	}

	if resp.Message != "" {
		return &Result{Message: resp.Message}, nil
	}

	return nil, ErrEmptyResponse
}
