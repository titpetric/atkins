package model

import "strings"

// RepositoryPolicy decides which repositories agents may build.
type RepositoryPolicy = string

// Repository policy values.
const (
	// PolicyOpen builds any repository a logged-in user dispatches.
	// It is the default: a personal instance has no third party to
	// defend against, and requiring a rule before the first run would
	// make the tool feel broken.
	PolicyOpen RepositoryPolicy = "open"

	// PolicyAllowlist builds only repositories matching an active
	// rule. With no rules configured, nothing runs — which is the
	// point of turning it on.
	PolicyAllowlist RepositoryPolicy = "allowlist"
)

// ValidRepositoryPolicy reports whether policy is a known value.
func ValidRepositoryPolicy(policy RepositoryPolicy) bool {
	return policy == PolicyOpen || policy == PolicyAllowlist
}

// MatchRepository reports whether a repository slug matches a pattern.
//
// Patterns are written against the normalized slug (`host/owner/name`):
//
//	github.com/titpetric/atkins   one repository
//	github.com/titpetric/*        every repository of one owner
//	github.com/**                 every repository on one host
//	**                            everything
//
// `*` matches within a path segment; `**` crosses segments. Matching is
// case-insensitive because slugs are lowercased.
func MatchRepository(pattern, slug string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	slug = strings.ToLower(strings.TrimSpace(slug))

	if pattern == "" || slug == "" {
		return false
	}

	return matchSegments(strings.Split(pattern, "/"), strings.Split(slug, "/"))
}

// matchSegments walks pattern and value segments together, recursing
// only where `**` forces a choice about how much to consume.
func matchSegments(pattern, value []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			// Trailing `**` swallows whatever is left, including
			// nothing at all.
			if len(pattern) == 1 {
				return true
			}
			for i := range len(value) + 1 {
				if matchSegments(pattern[1:], value[i:]) {
					return true
				}
			}
			return false
		}

		if len(value) == 0 {
			return false
		}
		if !matchSegment(pattern[0], value[0]) {
			return false
		}

		pattern, value = pattern[1:], value[1:]
	}

	return len(value) == 0
}

// matchSegment matches one path segment, where `*` stands for any run
// of characters that doesn't cross a separator.
func matchSegment(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")

	// The first and last parts are anchored; the rest may float.
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	value = value[len(parts[0]):]

	last := parts[len(parts)-1]
	if !strings.HasSuffix(value, last) {
		return false
	}
	value = value[:len(value)-len(last)]

	for _, part := range parts[1 : len(parts)-1] {
		index := strings.Index(value, part)
		if index < 0 {
			return false
		}
		value = value[index+len(part):]
	}

	return true
}
