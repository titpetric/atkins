package eventlog

import (
	"bytes"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	ulid "github.com/oklog/ulid/v2"
	yaml "gopkg.in/yaml.v3"
)

// Logger collects the events of one run and assembles the finished Log
// from them. A nil Logger records nothing. The package documentation
// says why it collects without a file.
type Logger struct {
	mu        sync.Mutex
	filePath  string
	metadata  RunMetadata
	events    []*Event
	startTime time.Time
	debug     bool
}

// NewLogger returns a logger of the run, which writes no file for an empty filePath.
func NewLogger(filePath, pipelineName, pipelineFile string, debug bool) *Logger {
	now := time.Now()
	runID := ulid.Make().String()

	metadata := RunMetadata{
		RunID:     runID,
		CreatedAt: now,
		Pipeline:  pipelineName,
		File:      pipelineFile,
	}

	// Capture git metadata
	metadata.Git = CaptureGitInfo()

	// Capture module path
	metadata.ModulePath = CaptureModulePath()

	return &Logger{
		filePath:  filePath,
		metadata:  metadata,
		events:    make([]*Event, 0),
		startTime: now,
		debug:     debug,
	}
}

// LogExec logs a single execution event (one per exec).
func (l *Logger) LogExec(result Result, id, run string, start float64, durationMs int64, err error) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	event := &Event{
		ID:       id,
		Type:     EventTypeStep,
		Start:    start,
		Duration: float64(durationMs) / 1000.0,
		Error:    errMsg,
		Run:      run,
		Result:   result,
	}
	if l.debug {
		event.GoroutineID = getGoroutineID()
	}
	l.events = append(l.events, event)
}

// LogCommand logs a command execution with full details.
func (l *Logger) LogCommand(entry LogEntry) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	event := &Event{
		ID:       entry.ID,
		Type:     entry.Type,
		Start:    entry.Start,
		Duration: float64(entry.DurationMs) / 1000.0,
		Error:    entry.Error,
		Command:  entry.Command,
		Dir:      entry.Dir,
		Output:   entry.Output,
		ExitCode: entry.ExitCode,
		ParentID: entry.ParentID,
	}
	if l.debug && len(entry.Env) > 0 {
		event.Env = entry.Env
	}
	l.events = append(l.events, event)
}

// elapsed returns seconds since the logger started.
func (l *Logger) elapsed() float64 {
	return time.Since(l.startTime).Seconds()
}

// Document returns nil on a nil Logger, else the finished log: the
// metadata, the state tree, every event and the summary.
func (l *Logger) Document(state *StateNode, summary *RunSummary) *Log {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	return &Log{
		Metadata: l.metadata,
		State:    state,
		Events:   l.events,
		Summary:  summary,
	}
}

// Write writes the final event log to the file. A logger with no file
// path writes nothing and reports no error.
func (l *Logger) Write(state *StateNode, summary *RunSummary) error {
	if l == nil || l.filePath == "" {
		return nil
	}

	data, err := yaml.Marshal(l.Document(state, summary))
	if err != nil {
		return err
	}

	return os.WriteFile(l.filePath, data, 0o644)
}

// GetStartTime returns the start time of the run.
func (l *Logger) GetStartTime() time.Time {
	if l == nil {
		return time.Time{}
	}
	return l.startTime
}

// GetElapsed returns the current elapsed time in seconds.
func (l *Logger) GetElapsed() float64 {
	if l == nil {
		return 0
	}
	return l.elapsed()
}

// GetEvents returns a copy of the events slice.
func (l *Logger) GetEvents() []*Event {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	events := make([]*Event, len(l.events))
	copy(events, l.events)
	return events
}

// getGoroutineID extracts the goroutine ID from the runtime stack.
// This is a common hack since Go doesn't expose goroutine IDs directly.
func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Stack output starts with "goroutine <id> ["
	fields := bytes.Fields(buf[:n])
	if len(fields) < 2 {
		return 0
	}
	id, _ := strconv.ParseUint(string(fields[1]), 10, 64)
	return id
}
