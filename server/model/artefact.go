package model

import (
	"strings"
	"unicode"
)

// DefaultContentType is the media type the server sends for an artefact
// when the agent could not tell what it uploaded.
const DefaultContentType = "application/octet-stream"

// MaxArtefactPathLength bounds the name the server stores an artefact under.
// The value is a path, not prose; anything longer is a mistake or an
// attempt to fill a column.
const MaxArtefactPathLength = 512

// ArtefactPath normalizes the name a pipeline gave a file into a clean
// path relative to the directory the job ran in, or "" when the name
// cannot mean that.
//
// This is the same decision cleanWorkingDirectory makes, for the same
// reason: the value arrives over the network and then addresses bytes on
// the server's disk. ArtefactPath rejects an absolute path or a `..`
// escape outright and never repairs it: a caller that names
// `../../etc/passwd` gets a refusal, and never a quiet `etc/passwd`.
func ArtefactPath(value string) string {
	return relativePath(value)
}

// ArtefactPattern normalizes one collection glob, or returns "".
//
// A pattern is a path with `*` and `**` in it, so the same rules apply
// as for a path: relative, no `..`, no leading separator.
func ArtefactPattern(value string) string {
	return relativePath(value)
}

// relativePath cleans a slash-separated path that must stay inside the
// directory it starts from.
func relativePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\`, "/"))
	if value == "" || len(value) > MaxArtefactPathLength || strings.HasPrefix(value, "/") {
		return ""
	}

	segments := strings.Split(value, "/")
	clean := make([]string, 0, len(segments))
	for _, segment := range segments {
		switch segment {
		case "", ".":
			// A doubled or trailing separator is sloppiness, not intent.
			continue
		case "..":
			return ""
		}
		if strings.ContainsFunc(segment, unicode.IsControl) {
			return ""
		}
		clean = append(clean, segment)
	}

	if len(clean) == 0 {
		return ""
	}
	return strings.Join(clean, "/")
}

// ArtefactContentType sanitizes a client-declared media type.
//
// The server echoes the value back on download, so a header the agent
// invented must not carry a newline into the response.
func ArtefactContentType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.ContainsFunc(value, unicode.IsControl) {
		return DefaultContentType
	}
	return value
}

// ArtefactPatterns parses the comma separated artefact_paths column
// into the globs an agent should collect, dropping anything that could
// not name a file inside the checkout.
//
// ArtefactPatterns checks patterns where the server reads them, on top
// of where it writes them: one version of the server can store a
// pattern and another can read it, and the agent is the one that touches
// the filesystem.
func ArtefactPatterns(value string) []string {
	parts := strings.Split(value, ",")

	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		if pattern := ArtefactPattern(part); pattern != "" {
			patterns = append(patterns, pattern)
		}
	}

	if len(patterns) == 0 {
		return nil
	}
	return patterns
}

// JoinArtefactPatterns renders patterns for the artefact_paths column,
// keeping only the ones that survive validation.
func JoinArtefactPatterns(patterns []string) string {
	return strings.Join(ArtefactPatterns(strings.Join(patterns, ",")), ",")
}

// MatchArtefactPath reports whether a path relative to the job's
// directory matches a collection pattern. `*` matches within a path
// segment, `**` across segments, exactly as the repository allowlist
// patterns do.
//
// Unlike MatchRepository this is case-sensitive: the server lowercases
// repository slugs on the way in and leaves filenames alone, and
// `*.JSON` collecting `scan.json` on one agent and not another would be
// worse than strict.
func MatchArtefactPath(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)

	if pattern == "" || value == "" {
		return false
	}

	return matchSegments(strings.Split(pattern, "/"), strings.Split(value, "/"))
}
