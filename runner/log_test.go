package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStepLogger_EmptyPath(t *testing.T) {
	logger, err := NewStepLogger("")
	assert.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Nil(t, logger.logger)
}

func TestNewStepLogger_CreatesFile(t *testing.T) {
	tmpFile := "test_atkins.log"
	defer os.Remove(tmpFile)

	logger, err := NewStepLogger(tmpFile)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.logger)

	// File should exist
	_, err = os.Stat(tmpFile)
	assert.NoError(t, err)
}

func TestStepLogger_LogRun(t *testing.T) {
	tmpFile := "test_run.log"
	defer os.Remove(tmpFile)

	logger, err := NewStepLoggerWithPipeline(tmpFile, "test-pipeline")
	require.NoError(t, err)

	logger.LogRun("test-job", 0, "test-step")

	// Verify log file exists
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "RUN")
	assert.Contains(t, string(content), "jobs.test-job.steps.0")
	assert.Contains(t, string(content), "test-pipeline")
}

func TestStepLogger_LogPass(t *testing.T) {
	tmpFile := "test_pass.log"
	defer os.Remove(tmpFile)

	logger, err := NewStepLoggerWithPipeline(tmpFile, "test-pipeline")
	require.NoError(t, err)

	logger.LogPass("test-job", 0, "test-step", 100)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "PASS")
	assert.Contains(t, string(content), "jobs.test-job.steps.0")
	assert.Contains(t, string(content), "test-pipeline")
	assert.Contains(t, string(content), "0.1000")
}

func TestStepLogger_LogFail(t *testing.T) {
	tmpFile := "test_fail.log"
	defer os.Remove(tmpFile)

	logger, err := NewStepLoggerWithPipeline(tmpFile, "test-pipeline")
	require.NoError(t, err)

	testErr := assert.AnError
	logger.LogFail("test-job", 0, "test-step", testErr, 150)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "FAIL")
	assert.Contains(t, string(content), "jobs.test-job.steps.0")
	assert.Contains(t, string(content), "test-pipeline")
	assert.Contains(t, string(content), "0.1500")
}

func TestStepLogger_LogSkip(t *testing.T) {
	tmpFile := "test_skip.log"
	defer os.Remove(tmpFile)

	logger, err := NewStepLoggerWithPipeline(tmpFile, "test-pipeline")
	require.NoError(t, err)

	logger.LogSkip("test-job", 0, "test-step")

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "SKIP")
	assert.Contains(t, string(content), "jobs.test-job.steps.0")
	assert.Contains(t, string(content), "test-pipeline")
}

func TestStepLogger_NilLogger(t *testing.T) {
	logger, err := NewStepLogger("")
	require.NoError(t, err)

	// These should not panic
	logger.LogRun("job", 0, "test")
	logger.LogPass("job", 0, "test", 100)
	logger.LogFail("job", 0, "test", nil, 100)
	logger.LogSkip("job", 0, "test")
}
