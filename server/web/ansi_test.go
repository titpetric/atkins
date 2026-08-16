package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

func TestAnsiSpansSplitsOnAppearance(t *testing.T) {
	spans := ansiSpans("plain \x1b[31mred\x1b[0m plain")

	require.Len(t, spans, 3)
	assert.Equal(t, LogSpan{Text: "plain "}, spans[0])
	assert.Equal(t, LogSpan{Text: "red", Class: "ansi-fg-red"}, spans[1])
	assert.Equal(t, LogSpan{Text: " plain"}, spans[2])
}

func TestAnsiSpansCarriesAppearanceForward(t *testing.T) {
	// atkins writes its tree as one attribute per sequence — bold, then
	// a colour — and expects both to hold until they are reset.
	spans := ansiSpans("\x1b[1m\x1b[37mheading\x1b[0m\ntail")

	require.Len(t, spans, 2)
	assert.Equal(t, "heading", spans[0].Text)
	assert.Equal(t, "ansi-fg-white ansi-bold", spans[0].Class)
	assert.Equal(t, "\ntail", spans[1].Text)
	assert.Empty(t, spans[1].Class)
}

func TestAnsiSpansHandlesTheRestOfSGR(t *testing.T) {
	tests := []struct {
		name    string
		content string
		class   string
	}{
		{"colour", "\x1b[32mx", "ansi-fg-green"},
		{"bright colour", "\x1b[92mx", "ansi-fg-bright-green"},
		{"background", "\x1b[41mx", "ansi-bg-red"},
		{"bright background", "\x1b[101mx", "ansi-bg-bright-red"},
		{"several at once", "\x1b[1;4;36mx", "ansi-fg-cyan ansi-bold ansi-underline"},
		{"dim", "\x1b[2mx", "ansi-dim"},
		{"italic", "\x1b[3mx", "ansi-italic"},
		{"colour then default", "\x1b[31;39mx", ""},
		{"bold then normal weight", "\x1b[1;22mx", ""},
		// An unknown parameter costs the colour it named, not the line:
		// a 256-colour tool renders as plain text rather than as an
		// unclosed span.
		{"256 colour", "\x1b[38;5;208mx", ""},
		{"bare reset", "\x1b[mx", ""},
		{"reset after a colour", "\x1b[31m\x1b[mx", ""},
		{"still coloured before the reset", "\x1b[31mx\x1b[0m", "ansi-fg-red"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spans := ansiSpans(test.content)
			require.NotEmpty(t, spans)
			assert.Equal(t, "x", spans[0].Text)
			assert.Equal(t, test.class, spans[0].Class)
		})
	}
}

func TestAnsiSpansDropsWhatAPageCannotHonour(t *testing.T) {
	// Cursor movement, line erasure and a terminal title are a
	// terminal's business. Printing them as text is the bug this
	// replaces; acting on them is not possible in a document.
	spans := ansiSpans("one\x1b[2K\x1b[1Atwo\x1b]0;title\x07three\x1bMfour")

	require.Len(t, spans, 1)
	assert.Equal(t, "onetwothreefour", spans[0].Text)
	assert.Empty(t, spans[0].Class)
}

func TestAnsiSpansDropsATruncatedSequence(t *testing.T) {
	// Output arrives in chunks, so the last one can end mid-escape. A
	// lone ESC helps nobody, and the rest arrives with the next flush.
	spans := ansiSpans("done\x1b[3")

	require.Len(t, spans, 1)
	assert.Equal(t, "done", spans[0].Text)
}

func TestAnsiSpansOnPlainOutput(t *testing.T) {
	spans := ansiSpans("no colour here\n")

	require.Len(t, spans, 1)
	assert.Equal(t, LogSpan{Text: "no colour here\n"}, spans[0])

	assert.Empty(t, ansiSpans(""))
}

func TestLogSpansJoinsEntries(t *testing.T) {
	// A colour set in one chunk of output holds into the next: the
	// entries are one stream that happened to be flushed in pieces.
	spans := logSpans([]model.JobLog{
		{Content: "\x1b[32mfirst"},
		{Content: " second\x1b[0m"},
	})

	require.Len(t, spans, 1)
	assert.Equal(t, "first second", spans[0].Text)
	assert.Equal(t, "ansi-fg-green", spans[0].Class)
}

func TestJobPageRendersColourAndStillEscapes(t *testing.T) {
	job := &model.Job{ID: "job-1", Status: model.JobStatusFailed, Command: "atkins"}

	rendered := render(t, jobView(&JobPage{
		Job: job,
		Log: []model.JobLog{{Content: "\x1b[1m\x1b[32mok\x1b[0m <script>alert(1)</script>\n"}},
	}, Links{}))

	assert.Contains(t, rendered, `<span class="ansi-fg-green ansi-bold">ok</span>`)
	// The escape sequences are gone from the text, rather than shown.
	assert.NotContains(t, rendered, "[1m")
	assert.NotContains(t, rendered, "\x1b")
	// And output is still output: colouring it changes nothing about
	// what a build is allowed to put on the page.
	assert.NotContains(t, rendered, "<script>alert(1)</script>")
	assert.Contains(t, rendered, "&lt;script&gt;")
}
