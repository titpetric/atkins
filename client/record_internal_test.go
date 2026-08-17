package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChunks covers splitting a transcript into log appends.
func TestChunks(t *testing.T) {
	t.Run("leaves a short transcript whole", func(t *testing.T) {
		assert.Equal(t, []string{"one line\n"}, chunks("one line\n", 32))
	})

	t.Run("reports nothing for nothing", func(t *testing.T) {
		assert.Empty(t, chunks("", 32))
	})

	t.Run("splits on a line boundary", func(t *testing.T) {
		parts := chunks("aaaa\nbbbb\ncccc\n", 10)

		// The cut lands after a newline rather than mid-line, so a log
		// entry is never half a line of somebody's build output.
		require.Len(t, parts, 2)
		assert.Equal(t, "aaaa\nbbbb\n", parts[0])
		assert.Equal(t, "cccc\n", parts[1])
	})

	t.Run("splits a line longer than the chunk", func(t *testing.T) {
		// One enormous line — minified output, a base64 blob — has no
		// boundary to find, and must still be shipped rather than held.
		parts := chunks(strings.Repeat("x", 25), 10)

		require.Len(t, parts, 3)
		assert.Equal(t, strings.Repeat("x", 10), parts[0])
		assert.Equal(t, strings.Repeat("x", 5), parts[2])
	})

	t.Run("loses nothing", func(t *testing.T) {
		transcript := strings.Repeat("a line of build output\n", 500)

		assert.Equal(t, transcript, strings.Join(chunks(transcript, logChunk), ""))
	})
}

// TestHeader covers the first thing a recorded job's page shows.
func TestHeader(t *testing.T) {
	t.Run("names the machine, the command and the commit", func(t *testing.T) {
		text := header(&Checkout{
			Revision: "0123456789abcdef0123456789abcdef01234567",
			Branch:   "main",
		}, "atkins test", "laptop")

		assert.Contains(t, text, "running locally on laptop")
		assert.Contains(t, text, "$ atkins test")
		assert.Contains(t, text, "commit: 0123456789ab (main)")
		assert.NotContains(t, text, "uncommitted")
	})

	t.Run("says when the log did not come from the commit it names", func(t *testing.T) {
		text := header(&Checkout{
			Revision: "0123456789abcdef0123456789abcdef01234567",
			Branch:   "main",
			Dirty:    true,
		}, "atkins test", "laptop")

		assert.Contains(t, text, "with uncommitted changes")
	})

	t.Run("names the directory a run happened in", func(t *testing.T) {
		text := header(&Checkout{WorkingDirectory: "server/api"}, "atkins test", "laptop")

		assert.Contains(t, text, "directory: server/api")
	})
}
