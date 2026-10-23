package runner

import (
	"context"
	"fmt"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/treeview"
)

// writeTranscript writes the finished tree to the caller's writer.
//
// It renders the same thing RenderFinal prints, colours included, so a
// job page reads like the terminal the run happened in, with no second,
// plainer account of it.
func (p *Pipeline) writeTranscript(root *treeview.Node) {
	if p.opts.Transcript == nil {
		return
	}

	// A transcript nobody could write is not worth interrupting a run
	// that has already finished.
	_, _ = fmt.Fprint(p.opts.Transcript, treeview.NewRenderer().Render(root))
}

// finishEventLog closes the run's log: it writes the file --log asked
// for and hands the same document to the recorder.
//
// Neither failure reaches the caller. The run has already happened and
// its exit code is already decided; a log that could not be written is
// not a reason to report a passing pipeline as failed.
func (p *Pipeline) finishEventLog(ctx context.Context, logger *eventlog.Logger, root *treeview.Node, runErr error) {
	if logger == nil {
		return
	}

	// Set root duration
	root.SetDuration(logger.GetElapsed())

	// Convert tree to state
	state := eventlog.NodeToStateNode(root)

	// Count steps and build summary
	total, passed, failed, skipped := eventlog.CountSteps(state)

	result := eventlog.ResultPass
	if runErr != nil || failed > 0 {
		result = eventlog.ResultFail
	}

	stats := eventlog.CaptureRuntimeStats()
	summary := &eventlog.RunSummary{
		Duration:     logger.GetElapsed(),
		TotalSteps:   total,
		PassedSteps:  passed,
		FailedSteps:  failed,
		SkippedSteps: skipped,
		Result:       result,
		MemoryAlloc:  stats.MemoryAlloc,
		Goroutines:   stats.Goroutines,
	}

	_ = logger.Write(state, summary)

	if p.opts.Recorder != nil {
		_ = p.opts.Recorder.Record(ctx, logger.Document(state, summary))
	}
}
