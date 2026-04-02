package agent

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// GitStats holds +/- line counts from git diff.
type GitStats struct {
	Added   int
	Removed int
}

// detectHostname returns the system hostname.
func detectHostname() string {
	out, err := exec.Command("uname", "-n").Output()
	if err != nil {
		if h, err := os.Hostname(); err == nil {
			return h
		}
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectGitBranch returns the current git branch name.
func detectGitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectGitStats returns line additions and deletions from git diff.
// Includes both staged and unstaged changes.
func detectGitStats(dir string) GitStats {
	var stats GitStats

	// Get unstaged changes
	cmd := exec.Command("git", "-C", dir, "diff", "--shortstat")
	if out, err := cmd.Output(); err == nil {
		parseGitShortstat(string(out), &stats)
	}

	// Get staged changes
	cmd = exec.Command("git", "-C", dir, "diff", "--cached", "--shortstat")
	if out, err := cmd.Output(); err == nil {
		parseGitShortstat(string(out), &stats)
	}

	return stats
}

// parseGitShortstat parses git shortstat output and adds to stats.
// Output format: " 3 files changed, 45 insertions(+), 12 deletions(-)"
func parseGitShortstat(output string, stats *GitStats) {
	parts := strings.Split(output, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			n, err := strconv.Atoi(fields[0])
			if err != nil {
				continue
			}
			if strings.Contains(fields[1], "insertion") {
				stats.Added += n
			} else if strings.Contains(fields[1], "deletion") {
				stats.Removed += n
			}
		}
	}
}
