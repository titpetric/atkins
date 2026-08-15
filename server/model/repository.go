package model

import "strings"

// RepositorySlug normalizes a git remote URL into a stable identity of
// the form `host/owner/name`. The slug is what the server deduplicates
// on, so `git@github.com:titpetric/atkins.git` and
// `https://github.com/titpetric/atkins` are one repository.
//
// An unrecognized remote is returned trimmed and lowercased rather than
// rejected; a private or self-hosted remote should still be usable
// without the server knowing its URL shape.
func RepositorySlug(remoteURL string) string {
	slug := strings.TrimSpace(remoteURL)
	if slug == "" {
		return ""
	}

	// scp-like syntax: git@host:owner/name.git
	if rest, ok := strings.CutPrefix(slug, "git@"); ok {
		slug = strings.Replace(rest, ":", "/", 1)
	}

	// URL syntax: scheme://[user@]host/owner/name.git
	if idx := strings.Index(slug, "://"); idx != -1 {
		slug = slug[idx+3:]
		if at := strings.Index(slug, "@"); at != -1 {
			slug = slug[at+1:]
		}
	}

	slug = strings.TrimSuffix(slug, "/")
	slug = strings.TrimSuffix(slug, ".git")

	return strings.ToLower(slug)
}
