package web

import (
	"strconv"
	"strings"

	"github.com/titpetric/atkins/server/model"
)

// A job's captured output is whatever the command wrote to a pipe, and
// atkins colours its tree. Written into a <pre> verbatim those colours
// are escape sequences a person has to read around, and a real build
// turns into noise.
//
// The bytes are kept as they arrived: the database holds what the
// command produced and `GET /api/job/{id}/log` hands it back raw, for
// a script that wants to re-colour it or strip it. The translation
// happens here, at the last moment, and emits classes rather than style
// attributes so the palette lives in one stylesheet with the rest of
// the page.

// LogSpan is a run of output that shares one appearance.
type LogSpan struct {
	Text string

	// Class is the space-separated class list for the run, empty for
	// output in the page's own colour.
	Class string
}

// ansiState is the appearance in force while parsing.
type ansiState struct {
	foreground string
	background string
	bold       bool
	dim        bool
	italic     bool
	underline  bool
}

// class renders the state as the class list a span carries.
func (s ansiState) class() string {
	classes := make([]string, 0, 6)

	if s.foreground != "" {
		classes = append(classes, "ansi-fg-"+s.foreground)
	}
	if s.background != "" {
		classes = append(classes, "ansi-bg-"+s.background)
	}
	if s.bold {
		classes = append(classes, "ansi-bold")
	}
	if s.dim {
		classes = append(classes, "ansi-dim")
	}
	if s.italic {
		classes = append(classes, "ansi-italic")
	}
	if s.underline {
		classes = append(classes, "ansi-underline")
	}

	return strings.Join(classes, " ")
}

// ansiColours maps an SGR colour parameter to the name its class uses.
var ansiColours = map[int]string{
	30: "black", 31: "red", 32: "green", 33: "yellow",
	34: "blue", 35: "magenta", 36: "cyan", 37: "white",
	90: "bright-black", 91: "bright-red", 92: "bright-green", 93: "bright-yellow",
	94: "bright-blue", 95: "bright-magenta", 96: "bright-cyan", 97: "bright-white",
}

// logSpans splits captured output into runs of one appearance.
//
// Only SGR sequences — the ones that set a colour or a weight — become
// spans. Everything else an escape sequence can ask for moves a cursor
// or clears a line, which a page that is not a terminal cannot honour
// and must not print either, so those are dropped.
func logSpans(entries []model.JobLog) []LogSpan {
	var raw strings.Builder
	for _, entry := range entries {
		raw.WriteString(entry.Content)
	}

	return ansiSpans(raw.String())
}

// ansiSpans is logSpans over one string.
func ansiSpans(content string) []LogSpan {
	var (
		spans   []LogSpan
		state   ansiState
		current strings.Builder
	)

	emit := func() {
		if current.Len() == 0 {
			return
		}
		spans = append(spans, LogSpan{Text: current.String(), Class: state.class()})
		current.Reset()
	}

	for i := 0; i < len(content); {
		if content[i] != 0x1b {
			current.WriteByte(content[i])
			i++
			continue
		}

		params, final, width, ok := escapeSequence(content[i:])
		if !ok {
			// A trailing escape with nothing after it yet: the next
			// chunk of output holds the rest, and printing a lone ESC
			// helps nobody.
			break
		}

		if final == 'm' {
			// The appearance changes here, so what came before it is a
			// span of its own.
			emit()
			state = applySGR(state, params)
		}

		i += width
	}

	emit()

	return spans
}

// escapeSequence measures the escape sequence at the front of content.
//
// It reports the parameters of a CSI sequence, its final byte, how many
// bytes to skip, and whether the sequence is complete. An OSC sequence
// (a terminal title, say) is measured and reported with a final byte
// that no caller acts on, which is how it gets dropped.
func escapeSequence(content string) (params string, final byte, width int, ok bool) {
	if len(content) < 2 {
		return "", 0, 0, false
	}

	switch content[1] {
	case '[':
		for i := 2; i < len(content); i++ {
			if c := content[i]; c >= 0x40 && c <= 0x7e {
				return content[2:i], c, i + 1, true
			}
		}
		return "", 0, 0, false

	case ']':
		// Ends at BEL, or at ST (ESC \).
		for i := 2; i < len(content); i++ {
			if content[i] == 0x07 {
				return "", 0, i + 1, true
			}
			if content[i] == 0x1b && i+1 < len(content) && content[i+1] == '\\' {
				return "", 0, i + 2, true
			}
		}
		return "", 0, 0, false
	}

	// A two-byte escape: nothing a page renders.
	return "", 0, 2, true
}

// applySGR folds one SGR sequence into the appearance.
//
// Unknown parameters are ignored rather than treated as a reset: an
// unrecognised colour should cost the colour, not the whole line.
func applySGR(state ansiState, params string) ansiState {
	if params == "" {
		// A bare ESC[m is ESC[0m.
		return ansiState{}
	}

	for _, field := range strings.Split(params, ";") {
		code, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			continue
		}

		switch {
		case code == 0:
			state = ansiState{}
		case code == 1:
			state.bold = true
		case code == 2:
			state.dim = true
		case code == 3:
			state.italic = true
		case code == 4:
			state.underline = true
		case code == 22:
			state.bold, state.dim = false, false
		case code == 23:
			state.italic = false
		case code == 24:
			state.underline = false
		case code == 39:
			state.foreground = ""
		case code == 49:
			state.background = ""
		default:
			if colour, found := ansiColours[code]; found {
				state.foreground = colour
				continue
			}
			if colour, found := ansiColours[code-10]; found {
				state.background = colour
			}
		}
	}

	return state
}
