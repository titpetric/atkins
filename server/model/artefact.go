package model

import (
	"strings"
	"unicode"
)

// DefaultContentType is what an artefact is served as when the agent
// could not tell what it uploaded.
const DefaultContentType = "application/octet-stream"

// MaxArtefactPathLength bounds the name an artefact is stored under.
// The value is a path, not prose; anything longer is a mistake or an
// attempt to fill a column.
const MaxArtefactPathLength = 512

// ArtefactPath normalizes the name a pipeline gave a file into a clean
// path relative to the directory the job ran in, or "" when the name
// cannot mean that.
//
// This is the same decision cleanWorkingDirectory makes, and it is made
// here for the same reason: the value arrives over the network and ends
// up addressing bytes on the server's disk. An absolute path or a `..`
// escape is rejected outright rather than repaired, because a caller
// asking for `../../etc/passwd` should be told no, not quietly handed
// `etc/passwd`.
func ArtefactPath(value string) string {
	return relativePath(value)
}

// ArtefactPattern normalizes one collection glob, or returns "".
//
// A pattern is a path with `*` and `**` in it, so it is held to exactly
// the same rules as a path: relative, no `..`, no leading separator.
func ArtefactPattern(value string) string {
	return relativePath(value)
}

// relativePath cleans a slash-separated path that must stay inside the
// directory it is relative to.
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
// It is echoed back on download, so a header the agent invented must
// not be able to carry a newline into the response.
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
// Patterns are validated where they are read rather than only where
// they are written: a pattern can be stored by one version of the
// server and read by another, and the agent is the one that touches the
// filesystem.
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
// Unlike MatchRepository this is case-sensitive: repository slugs are
// lowercased on the way in, filenames are not, and `*.JSON` collecting
// `scan.json` on one agent and not another would be worse than strict.
func MatchArtefactPath(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)

	if pattern == "" || value == "" {
		return false
	}

	return matchSegments(strings.Split(pattern, "/"), strings.Split(value, "/"))
}
