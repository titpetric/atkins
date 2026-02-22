package eventlog

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestNewLogger_NilWhenEmpty(t *testing.T) {
	logger := NewLogger("", "test-pipeline", "test.yml", false)
	assert.Nil(t, logger)
}

func TestNewLogger_CreatesLogger(t *testing.T) {
	tmpFile := "test_eventlog.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)
	assert.NotEmpty(t, logger.metadata.RunID)
	assert.Equal(t, "test-pipeline", logger.metadata.Pipeline)
	assert.Equal(t, "test.yml", logger.metadata.File)
}

func TestLogger_LogExec_Pass(t *testing.T) {
	tmpFile := "test_pass.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogExec(ResultPass, "jobs.test-job.steps.0", "echo hello", 0.5, 100, nil)

	events := logger.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "jobs.test-job.steps.0", events[0].ID)
	assert.Equal(t, "echo hello", events[0].Run)
	assert.Equal(t, ResultPass, events[0].Result)
	assert.Equal(t, 0.5, events[0].Start)
	assert.Equal(t, 0.1, events[0].Duration)
	assert.Empty(t, events[0].Error)
}

func TestLogger_LogExec_Fail(t *testing.T) {
	tmpFile := "test_fail.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	testErr := assert.AnError
	logger.LogExec(ResultFail, "jobs.test-job.steps.0", "bad command", 1.0, 150, testErr)

	events := logger.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "jobs.test-job.steps.0", events[0].ID)
	assert.Equal(t, "bad command", events[0].Run)
	assert.Equal(t, ResultFail, events[0].Result)
	assert.Equal(t, 1.0, events[0].Start)
	assert.Equal(t, 0.15, events[0].Duration)
	assert.Contains(t, events[0].Error, "assert.AnError")
}

func TestLogger_LogExec_Skip(t *testing.T) {
	tmpFile := "test_skip.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogExec(ResultSkipped, "jobs.test-job.steps.0", "skipped step", 2.0, 0, nil)

	events := logger.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "jobs.test-job.steps.0", events[0].ID)
	assert.Equal(t, "skipped step", events[0].Run)
	assert.Equal(t, ResultSkipped, events[0].Result)
	assert.Equal(t, 2.0, events[0].Start)
	assert.Equal(t, 0.0, events[0].Duration)
}

func TestLogger_NilSafe(t *testing.T) {
	var logger *Logger

	// All methods should be safe to call on nil
	logger.LogExec(ResultPass, "id", "run", 0, 100, nil)
	assert.Nil(t, logger.GetEvents())
	assert.Equal(t, float64(0), logger.GetElapsed())
	assert.Equal(t, time.Time{}, logger.GetStartTime())
	assert.NoError(t, logger.Write(nil, nil))
}

func TestLogger_Write(t *testing.T) {
	tmpFile := "test_write.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogExec(ResultPass, "jobs.test-job.steps.0", "echo hello", 0.1, 100, nil)

	state := &StateNode{
		Name:      "test-pipeline",
		Status:    "passed",
		Result:    ResultPass,
		CreatedAt: time.Now(),
		Duration:  0.5,
	}

	summary := &RunSummary{
		Duration:     0.5,
		TotalSteps:   1,
		PassedSteps:  1,
		FailedSteps:  0,
		SkippedSteps: 0,
		Result:       ResultPass,
	}

	err := logger.Write(state, summary)
	require.NoError(t, err)

	// Read and verify the file
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var log Log
	err = yaml.Unmarshal(data, &log)
	require.NoError(t, err)

	assert.NotEmpty(t, log.Metadata.RunID)
	assert.Equal(t, "test-pipeline", log.Metadata.Pipeline)
	assert.Equal(t, "test-pipeline", log.State.Name)
	assert.Len(t, log.Events, 1)
	assert.Equal(t, ResultPass, log.Summary.Result)
}

func TestLogger_GetElapsed(t *testing.T) {
	tmpFile := "test_elapsed.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	time.Sleep(10 * time.Millisecond)

	elapsed := logger.GetElapsed()
	assert.Greater(t, elapsed, 0.0)
}

func TestLogger_DebugGoroutineID(t *testing.T) {
	tmpFile := "test_debug.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", true)
	require.NotNil(t, logger)

	logger.LogExec(ResultPass, "jobs.test-job.steps.0", "echo hello", 0.1, 100, nil)

	events := logger.GetEvents()
	require.Len(t, events, 1)

	// Goroutine ID should be non-zero when debug is enabled
	assert.Greater(t, events[0].GoroutineID, uint64(0))
}

func TestLogger_NoGoroutineIDWithoutDebug(t *testing.T) {
	tmpFile := "test_nodebug.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile) // ignore error, file may not exist
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogExec(ResultPass, "jobs.test-job.steps.0", "echo hello", 0.1, 100, nil)

	events := logger.GetEvents()
	require.Len(t, events, 1)

	// Goroutine ID should be zero when debug is disabled
	assert.Equal(t, uint64(0), events[0].GoroutineID)
}

func TestGetGoroutineID(t *testing.T) {
	id := getGoroutineID()
	assert.Greater(t, id, uint64(0))
}

func TestLogger_LogCommand_Step(t *testing.T) {
	tmpFile := "test_command_step.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile)
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogCommand(LogEntry{
		Type:       EventTypeStep,
		ID:         "jobs.test.steps.0",
		Command:    "echo hello",
		Dir:        "/tmp",
		Output:     "hello\n",
		Start:      0.5,
		DurationMs: 100,
	})

	require.Len(t, logger.events, 1)
	cmd := logger.events[0]
	assert.Equal(t, "jobs.test.steps.0", cmd.ID)
	assert.Equal(t, EventTypeStep, cmd.Type)
	assert.Equal(t, "echo hello", cmd.Command)
	assert.Equal(t, "/tmp", cmd.Dir)
	assert.Equal(t, "hello\n", cmd.Output)
	assert.Empty(t, cmd.Error)
	assert.Equal(t, 0, cmd.ExitCode)
	assert.Equal(t, 0.5, cmd.Start)
	assert.Equal(t, 0.1, cmd.Duration)
}

func TestLogger_LogCommand_Substitution(t *testing.T) {
	tmpFile := "test_command_subst.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile)
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogCommand(LogEntry{
		Type:       EventTypeSubstitution,
		ID:         "subst-12345",
		ParentID:   "jobs.test.steps.0",
		Command:    "date +%Y",
		Dir:        "/home/user",
		Output:     "2024",
		Start:      1.0,
		DurationMs: 50,
	})

	require.Len(t, logger.events, 1)
	cmd := logger.events[0]
	assert.Equal(t, "subst-12345", cmd.ID)
	assert.Equal(t, EventTypeSubstitution, cmd.Type)
	assert.Equal(t, "date +%Y", cmd.Command)
	assert.Equal(t, "jobs.test.steps.0", cmd.ParentID)
	assert.Equal(t, "2024", cmd.Output)
}

func TestLogger_LogCommand_WithError(t *testing.T) {
	tmpFile := "test_command_error.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile)
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogCommand(LogEntry{
		Type:       EventTypeStep,
		ID:         "jobs.test.steps.0",
		Command:    "exit 1",
		Dir:        "/tmp",
		Error:      "command failed",
		ExitCode:   1,
		Start:      0.5,
		DurationMs: 100,
	})

	require.Len(t, logger.events, 1)
	cmd := logger.events[0]
	assert.Equal(t, 1, cmd.ExitCode)
	assert.Equal(t, "command failed", cmd.Error)
}

func TestLogger_LogCommand_WithEnvDebug(t *testing.T) {
	tmpFile := "test_command_env.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile)
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", true)
	require.NotNil(t, logger)

	env := []string{"FOO=bar", "BAZ=qux"}
	logger.LogCommand(LogEntry{
		Type:       EventTypeStep,
		ID:         "jobs.test.steps.0",
		Command:    "env",
		Dir:        "/tmp",
		Output:     "FOO=bar\nBAZ=qux\n",
		Start:      0.5,
		DurationMs: 100,
		Env:        env,
	})

	require.Len(t, logger.events, 1)
	cmd := logger.events[0]
	assert.Equal(t, env, cmd.Env)
}

func TestLogger_LogCommand_NoEnvWithoutDebug(t *testing.T) {
	tmpFile := "test_command_noenv.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile)
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	env := []string{"FOO=bar"}
	logger.LogCommand(LogEntry{
		Type:       EventTypeStep,
		ID:         "jobs.test.steps.0",
		Command:    "env",
		Dir:        "/tmp",
		Start:      0.5,
		DurationMs: 100,
		Env:        env,
	})

	require.Len(t, logger.events, 1)
	cmd := logger.events[0]
	assert.Nil(t, cmd.Env)
}

func TestLogger_LogCommand_NilSafe(t *testing.T) {
	var logger *Logger
	// Should not panic
	logger.LogCommand(LogEntry{
		Type:    EventTypeStep,
		ID:      "id",
		Command: "cmd",
		Dir:     "/tmp",
	})
}

func TestLogger_Write_WithCommands(t *testing.T) {
	tmpFile := "test_write_commands.yml"
	t.Cleanup(func() {
		_ = os.Remove(tmpFile)
	})

	logger := NewLogger(tmpFile, "test-pipeline", "test.yml", false)
	require.NotNil(t, logger)

	logger.LogExec(ResultPass, "jobs.test.steps.0", "step name", 0.1, 100, nil)
	logger.LogCommand(LogEntry{
		Type:       EventTypeStep,
		ID:         "jobs.test.steps.0",
		Command:    "echo test",
		Dir:        "/tmp",
		Output:     "test\n",
		Start:      0.1,
		DurationMs: 100,
	})
	logger.LogCommand(LogEntry{
		Type:       EventTypeSubstitution,
		ID:         "subst-1",
		ParentID:   "jobs.test.steps.0",
		Command:    "date",
		Dir:        "/tmp",
		Output:     "2024",
		Start:      0.05,
		DurationMs: 10,
	})

	state := &StateNode{
		Name:      "test-pipeline",
		Status:    "passed",
		Result:    ResultPass,
		CreatedAt: time.Now(),
	}

	err := logger.Write(state, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var log Log
	err = yaml.Unmarshal(data, &log)
	require.NoError(t, err)

	assert.Len(t, log.Events, 3)
	assert.Equal(t, EventTypeStep, log.Events[0].Type)
	assert.Equal(t, EventTypeStep, log.Events[1].Type)
	assert.Equal(t, EventTypeSubstitution, log.Events[2].Type)
}
