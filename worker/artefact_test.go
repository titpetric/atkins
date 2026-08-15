package worker

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/server/model"
)

// collecting returns a jobContext that also declares artefact patterns.
func collecting(remote, workingDirectory, command, patterns string) *jobContext {
	job := job(remote, workingDirectory, command)
	job.Job.ArtefactPaths = patterns
	return job
}

func TestCollectUploadsTheOutputDirectory(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	// The zero-configuration declaration: whatever the job copies into
	// $ATKINS_ARTEFACTS is kept.
	worker.run(t.Context(), job(remote, "",
		`mkdir -p "$ATKINS_ARTEFACTS/reports" && echo '{"findings":3}' > "$ATKINS_ARTEFACTS/reports/scan.json"`))

	_, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status)

	uploaded := fake.stored()
	require.Contains(t, uploaded, "reports/scan.json")
	assert.JSONEq(t, `{"findings":3}`, string(uploaded["reports/scan.json"]))
}

func TestCollectMatchesDeclaredPatterns(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	// A pipeline that knows nothing about artefacts: the files are
	// written where the command already writes them, and the job says
	// what to pick up.
	worker.run(t.Context(), collecting(remote, "",
		`echo '{"ok":true}' > coverage.json; mkdir -p out; echo report > out/summary.txt; echo noise > noise.log`,
		"*.json,out/*.txt"))

	_, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status)

	uploaded := fake.stored()
	assert.Contains(t, uploaded, "coverage.json")
	assert.Contains(t, uploaded, "out/summary.txt")
	assert.NotContains(t, uploaded, "noise.log")

	// The checkout's own object store is never an output.
	for path := range uploaded {
		assert.NotContains(t, path, ".git/")
	}
}

func TestCollectRunsAfterAFailure(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	// The artefacts of a failing run are the ones worth having, so
	// collection is not conditional on the exit code.
	worker.run(t.Context(), collecting(remote, "",
		`echo 'stack trace' > crash.log; exit 3`, "*.log"))

	_, status := fake.result()
	require.Equal(t, client.StatusFailed, status.Status)
	assert.Equal(t, int64(3), status.ExitCode)

	uploaded := fake.stored()
	require.Contains(t, uploaded, "crash.log")
	assert.Contains(t, string(uploaded["crash.log"]), "stack trace")
}

func TestCollectRunsAfterATimeout(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)
	worker.opts.JobTimeout = 500 * time.Millisecond

	worker.run(t.Context(), collecting(remote, "",
		`echo 'partial' > progress.log; sleep 30`, "*.log"))

	_, status := fake.result()
	require.Equal(t, client.StatusTimeout, status.Status)

	// The job's context has expired by now: collection must not be
	// cancelled with it, or a timeout would lose exactly the output
	// that explains it.
	assert.Contains(t, fake.stored(), "progress.log")
}

func TestCollectStaysInsideTheWorkspace(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	secret := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("not yours"), 0o600))

	// A traversal pattern, and a symlink pointing out of the checkout.
	// Collection walks the tree rather than expanding patterns against
	// the filesystem, so neither can name a file outside it.
	worker.run(t.Context(), collecting(remote, "",
		"ln -s "+secret+" link.txt; echo mine > kept.txt",
		"../../*,/etc/*,**/*.txt"))

	_, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status)

	uploaded := fake.stored()
	assert.Contains(t, uploaded, "kept.txt")
	assert.NotContains(t, uploaded, "link.txt")
	for _, content := range uploaded {
		assert.NotContains(t, string(content), "not yours")
	}
}

func TestCollectUploadsNothingWhenNothingWasProduced(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), collecting(remote, "", "echo hello", "*.json"))

	_, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status)
	assert.Empty(t, fake.stored())
}

func TestCollectCleansUpTheOutputDirectory(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", `echo kept > "$ATKINS_ARTEFACTS/kept.txt"`))

	require.Contains(t, fake.stored(), "kept.txt")

	// The staged copy would fill the agent's disk if it outlived the
	// job, exactly as the work tree would.
	entries, err := os.ReadDir(filepath.Join(worker.opts.DataDir, "work"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestCollectReportsAFailedUpload(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	// The server refuses everything: a rejected artefact is reported in
	// the job's output rather than swallowed, and it does not change
	// the outcome of the command.
	fake.refuseArtefacts = true

	worker.run(t.Context(), collecting(remote, "", "echo mine > kept.txt", "*.txt"))

	output, status := fake.result()
	assert.Equal(t, client.StatusPassed, status.Status)
	assert.Contains(t, output, "kept.txt was not stored")
}

func TestContentTypeFromExtension(t *testing.T) {
	assert.Contains(t, contentType("reports/scan.json"), "application/json")
	// Unrecognized is left empty; the server applies its own default
	// rather than the agent guessing wrong.
	assert.Empty(t, contentType("core.dump-42"))
}
