package eventlog

import (
	"os/exec"
	"strings"
)

// CaptureGitInfo captures git repository information.
func CaptureGitInfo() *GitInfo {
	info := &GitInfo{}

	// Get commit
	if out, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		info.Commit = strings.TrimSpace(string(out))
	}

	// Get branch
	if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		info.Branch = strings.TrimSpace(string(out))
	}

	// Get remote URL
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		info.RemoteURL = strings.TrimSpace(string(out))
		info.Repository = extractRepoFromURL(info.RemoteURL)
	}

	// Return nil if no git info was captured
	if info.Commit == "" && info.Branch == "" && info.RemoteURL == "" {
		return nil
	}

	return info
}

// extractRepoFromURL extracts repository name from a git URL.
func extractRepoFromURL(url string) string {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ".git")

	return url
}
