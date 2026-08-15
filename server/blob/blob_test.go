package blob

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDir returns a store over a temporary root.
func testDir(t *testing.T) *Dir {
	t.Helper()

	store := NewDir(filepath.Join(t.TempDir(), "artefacts"))
	require.NoError(t, store.Prepare())

	return store
}

// read drains a reader and closes it.
func read(t *testing.T, contents io.ReadCloser) []byte {
	t.Helper()

	defer contents.Close()

	body, err := io.ReadAll(contents)
	require.NoError(t, err)

	return body
}

func TestPutAndOpen(t *testing.T) {
	store := testDir(t)
	content := []byte(`{"findings": 3}`)

	written, err := store.Put(t.Context(), Key("job-1", "artefact-1"), bytes.NewReader(content), 0)
	require.NoError(t, err)

	assert.Equal(t, int64(len(content)), written.Size)

	// The checksum is computed while writing rather than by reading the
	// file back.
	sum := sha256.Sum256(content)
	assert.Equal(t, hex.EncodeToString(sum[:]), written.Checksum)

	contents, err := store.Open(t.Context(), Key("job-1", "artefact-1"))
	require.NoError(t, err)
	assert.Equal(t, content, read(t, contents))
}

func TestPutStopsAtTheLimit(t *testing.T) {
	store := testDir(t)

	_, err := store.Put(t.Context(), Key("job-1", "artefact-1"),
		bytes.NewReader(bytes.Repeat([]byte("x"), 65)), 64)
	assert.ErrorIs(t, err, ErrTooLarge)

	// Nothing readable, and no partial file left in its place: a
	// refused upload must not become a truncated artefact.
	_, err = store.Open(t.Context(), Key("job-1", "artefact-1"))
	assert.Error(t, err)

	assert.Empty(t, leftovers(t, store.Root()))

	// Exactly at the limit is stored.
	written, err := store.Put(t.Context(), Key("job-1", "artefact-2"),
		bytes.NewReader(bytes.Repeat([]byte("x"), 64)), 64)
	require.NoError(t, err)
	assert.Equal(t, int64(64), written.Size)
}

func TestOpenMissingKey(t *testing.T) {
	store := testDir(t)

	_, err := store.Open(t.Context(), Key("job-1", "nothing"))
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveIsIdempotent(t *testing.T) {
	store := testDir(t)

	key := Key("job-1", "artefact-1")
	_, err := store.Put(t.Context(), key, strings.NewReader("contents"), 0)
	require.NoError(t, err)

	require.NoError(t, store.Remove(t.Context(), key))

	// Retention runs on a ticker and may see the same row twice;
	// removing what is already gone is not a failure.
	require.NoError(t, store.Remove(t.Context(), key))

	// The job's directory goes with its last artefact.
	assert.NoDirExists(t, filepath.Join(store.Root(), "job-1"))
}

func TestKeysStayUnderTheRoot(t *testing.T) {
	store := testDir(t)

	for _, key := range []string{"", "   ", "/etc/passwd", "../escape", "job-1/../../escape", ".."} {
		_, err := store.Put(t.Context(), key, strings.NewReader("planted"), 0)
		assert.ErrorIs(t, err, ErrInvalidKey, "key %q", key)

		_, err = store.Open(t.Context(), key)
		assert.ErrorIs(t, err, ErrInvalidKey, "key %q", key)

		assert.ErrorIs(t, store.Remove(t.Context(), key), ErrInvalidKey, "key %q", key)
	}
}

func TestPutOverwritesTheSameKey(t *testing.T) {
	store := testDir(t)
	key := Key("job-1", "artefact-1")

	_, err := store.Put(t.Context(), key, strings.NewReader("first"), 0)
	require.NoError(t, err)

	_, err = store.Put(t.Context(), key, strings.NewReader("second"), 0)
	require.NoError(t, err)

	contents, err := store.Open(t.Context(), key)
	require.NoError(t, err)
	assert.Equal(t, []byte("second"), read(t, contents))

	assert.Empty(t, leftovers(t, store.Root()))
}

func TestPutReportsAFailedRead(t *testing.T) {
	store := testDir(t)

	_, err := store.Put(t.Context(), Key("job-1", "artefact-1"), failingReader{}, 0)
	assert.Error(t, err)

	// A connection that died mid-upload leaves no debris either.
	assert.Empty(t, leftovers(t, store.Root()))
}

// failingReader stands in for an upload whose connection dropped.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

// Verify contract: a context is accepted but unused by the filesystem
// backend, so a cancelled one must not be mistaken for a failure.
func TestContextIsNotRequired(t *testing.T) {
	store := testDir(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := store.Put(ctx, Key("job-1", "artefact-1"), strings.NewReader("contents"), 0)
	assert.NoError(t, err)
}

// leftovers lists the temporary files a store has left behind.
func leftovers(t *testing.T, root string) []string {
	t.Helper()

	var found []string
	require.NoError(t, filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(name, ".part") {
			found = append(found, name)
		}
		return nil
	}))

	return found
}
