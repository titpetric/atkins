package client

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotARepository is returned when a directory is not inside a git
// work tree. Dispatch is skipped in that case: without a repository
// there is nothing for another machine to check out.
var ErrNotARepository = errors.New("not inside a git repository")

// Errors reported when a checkout cannot be reproduced elsewhere.
var (
	// ErrDirtyCheckout is a work tree with uncommitted changes.
	ErrDirtyCheckout = errors.New("the working tree has uncommitted changes")

	// ErrUnpushedCheckout is a commit no remote has.
	ErrUnpushedCheckout = errors.New("HEAD has not been pushed to a remote")
)

// Checkout is what the client reports about the working copy a run
// happens in. It is the whole of what the server needs to reproduce the
// run elsewhere: which repository, where inside it, at what revision.
type Checkout struct {
	// Root is the absolute path of the repository work tree.
	Root string

	// RemoteURL is the `origin` remote, or the first remote defined.
	RemoteURL string

	// WorkingDirectory is the run directory relative to Root. Empty
	// when the run happens at the repository root.
	WorkingDirectory string

	Branch        string
	Revision      string
	DefaultBranch string

	// Dirty reports uncommitted changes in the work tree, tracked or
	// not. A local run of a dirty tree is ordinary; a dispatched one
	// would build something else.
	Dirty bool

	// Unpushed reports that no remote has HEAD, so nothing can clone it.
	Unpushed bool
}

// Publishable reports whether another machine could reproduce this
// checkout from the repository's remote.
//
// A dispatched run names the commit it was started from, so a tree that
// only exists on this disk queues a job that fails in `git checkout`
// minutes later. The refusal belongs here, where the fix — commit and
// push, or run locally — is still in front of the person.
func (c *Checkout) Publishable() error {
	switch {
	case c.Dirty:
		return ErrDirtyCheckout
	case c.Unpushed:
		return ErrUnpushedCheckout
	}
	return nil
}

// Payload converts the checkout into a dispatch payload.
//
// The ref is the commit rather than the branch. A dispatched run belongs
// to the code in front of the person who started it, and a branch name
// would let the agent build whatever that branch has moved to by the
// time the job is claimed.
func (c *Checkout) Payload() RepositoryPayload {
	ref := c.Revision
	if ref == "" {
		ref = c.Branch
	}

	return RepositoryPayload{
		RemoteURL:     c.RemoteURL,
		Ref:           ref,
		DefaultBranch: c.DefaultBranch,
	}
}

// DetectCheckout inspects the git work tree containing dir.
//
// A repository with no remote is rejected along with a plain directory:
// the point of dispatch is that some other machine can fetch the same
// code, and a purely local repository fails that test.
func DetectCheckout(dir string) (*Checkout, error) {
	root := git(dir, "rev-parse", "--show-toplevel")
	if root == "" {
		return nil, ErrNotARepository
	}

	checkout := &Checkout{
		Root:          root,
		RemoteURL:     detectRemoteURL(dir),
		Branch:        git(dir, "rev-parse", "--abbrev-ref", "HEAD"),
		Revision:      git(dir, "rev-parse", "HEAD"),
		DefaultBranch: detectDefaultBranch(dir),
		Dirty:         git(dir, "status", "--porcelain") != "",
		Unpushed:      detectUnpushed(dir),
	}
	if checkout.RemoteURL == "" {
		return nil, ErrNotARepository
	}

	if relative, err := filepath.Rel(root, absolute(dir)); err == nil && relative != "." {
		checkout.WorkingDirectory = filepath.ToSlash(relative)
	}

	return checkout, nil
}

// detectRemoteURL prefers `origin` and falls back to the first remote.
func detectRemoteURL(dir string) string {
	if url := git(dir, "remote", "get-url", "origin"); url != "" {
		return url
	}

	remotes := git(dir, "remote")
	for _, remote := range strings.Fields(remotes) {
		if url := git(dir, "remote", "get-url", remote); url != "" {
			return url
		}
	}

	return ""
}

// detectUnpushed reports whether HEAD is missing from every remote.
//
// The question is which commit, not which branch: a commit reachable
// from any remote ref can be cloned, whatever the local branch is
// called and whether or not it tracks anything.
func detectUnpushed(dir string) bool {
	// A repository with no commits has nothing to push and nothing to
	// dispatch; the empty revision is what reports that.
	if git(dir, "rev-parse", "HEAD") == "" {
		return false
	}
	return git(dir, "rev-list", "-n", "1", "HEAD", "--not", "--remotes") != ""
}

// detectDefaultBranch reads origin's HEAD, e.g. `origin/main`. It is
// advisory: a checkout without a cached remote HEAD reports nothing.
func detectDefaultBranch(dir string) string {
	head := git(dir, "rev-parse", "--abbrev-ref", "origin/HEAD")
	if head == "" {
		return ""
	}
	return strings.TrimPrefix(head, "origin/")
}

// git runs a git command in dir and returns its trimmed stdout, or an
// empty string for anything that fails.
func git(dir string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// absolute resolves dir, falling back to the input when it can't.
func absolute(dir string) string {
	if resolved, err := filepath.Abs(dir); err == nil {
		return resolved
	}
	return dir
}
