package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// Workspace is a checkout prepared for one job.
type Workspace struct {
	// Root is the work tree the job runs under.
	Root string

	// Dir is the directory the command runs in: Root joined with the
	// job's working directory.
	Dir string
}

// prepare produces a checkout for a job.
//
// Clones are cached per repository under <DataDir>/repos, so the second
// job for a repository fetches instead of downloading it again. Each
// job then gets its own work tree cloned from that cache, which keeps
// concurrent jobs from fighting over one checkout.
func (w *Worker) prepare(ctx context.Context, job *jobContext) (*Workspace, error) {
	cache, err := w.cacheRepository(ctx, job)
	if err != nil {
		return nil, err
	}

	root := filepath.Join(w.opts.DataDir, "work", job.Job.ID)
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}

	// A local clone of the mirror is cheap: git hardlinks the objects.
	if _, err := w.git(ctx, "", "clone", "--no-checkout", cache, root); err != nil {
		return nil, fmt.Errorf("clone work tree: %w", err)
	}

	if err := w.checkout(ctx, root, job); err != nil {
		return nil, err
	}

	dir := root
	if relative := CleanWorkingDirectory(job.Job.WorkingDirectory); relative != "" {
		dir = filepath.Join(root, filepath.FromSlash(relative))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("working directory %q does not exist in the repository", relative)
		}
	}

	return &Workspace{Root: root, Dir: dir}, nil
}

// CleanWorkingDirectory normalizes a job's working directory into a
// clean path relative to the repository root, or "" for the root.
//
// The server sanitizes this before storing it, so this is the second
// gate on the same decision. It is worth having: the value arrives over
// the network and is about to become a directory the agent runs a shell
// command in, and an agent should not be one server bug away from
// executing outside its checkout.
func CleanWorkingDirectory(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." || dir == "/" {
		return ""
	}

	dir = path.Clean(strings.TrimPrefix(strings.ReplaceAll(dir, `\`, "/"), "/"))
	if dir == "." || dir == ".." || strings.HasPrefix(dir, "../") {
		return ""
	}

	return dir
}

// cacheRepository clones or updates the mirror for a repository and
// returns its path.
func (w *Worker) cacheRepository(ctx context.Context, job *jobContext) (string, error) {
	remote := CloneURL(job.Repository.RemoteURL, job.Repository.Slug, w.opts.PreferHTTPS)
	if remote == "" {
		return "", fmt.Errorf("job %s has no repository remote", job.Job.ID)
	}

	cache := filepath.Join(w.opts.DataDir, "repos", filepath.FromSlash(cachePath(job.Repository.Slug)))

	if _, err := os.Stat(filepath.Join(cache, "HEAD")); err == nil {
		// Known repository: refresh it in place.
		if out, err := w.git(ctx, cache, "remote", "update", "--prune"); err != nil {
			return "", fmt.Errorf("update %s: %w\n%s", remote, err, out)
		}
		return cache, nil
	}

	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		return "", fmt.Errorf("create cache directory: %w", err)
	}

	// A mirror carries every ref, so a later job can check out a tag or
	// a branch that didn't exist when the cache was created.
	if out, err := w.git(ctx, "", "clone", "--mirror", remote, cache); err != nil {
		// Leave nothing half-cloned behind, or the next job would
		// mistake the debris for a usable cache.
		_ = os.RemoveAll(cache)
		return "", fmt.Errorf("clone %s: %w\n%s", remote, err, out)
	}

	return cache, nil
}

// checkout moves the work tree to the revision the job named, falling
// back to its branch and then to whatever the mirror's HEAD points at.
func (w *Worker) checkout(ctx context.Context, root string, job *jobContext) error {
	candidates := []string{}
	if job.Job.Revision != "" {
		candidates = append(candidates, job.Job.Revision)
	}
	if job.Job.Branch != "" && job.Job.Branch != "HEAD" {
		candidates = append(candidates, "origin/"+job.Job.Branch, job.Job.Branch)
	}
	if job.Repository.DefaultBranch != "" {
		candidates = append(candidates, "origin/"+job.Repository.DefaultBranch)
	}
	candidates = append(candidates, "HEAD")

	var lastErr error
	for _, ref := range candidates {
		if out, err := w.git(ctx, root, "checkout", "--force", ref); err == nil {
			return nil
		} else {
			lastErr = fmt.Errorf("checkout %s: %w\n%s", ref, err, out)
		}
	}

	return lastErr
}

// cachePath turns a repository slug into a relative directory path.
// Slugs are `host/owner/name`, which nests naturally; anything odd is
// flattened so it cannot escape the cache root.
func cachePath(slug string) string {
	clean := filepath.ToSlash(filepath.Clean("/" + strings.TrimSpace(slug)))
	return strings.TrimPrefix(clean, "/") + ".git"
}

// CloneURL decides what to hand to `git clone`.
//
// With preferHTTPS, an ssh remote is rewritten to https using the slug
// the server derived. A container has no key agent, so the ssh form
// would fail on a repository that clones fine anonymously over https.
func CloneURL(remoteURL, slug string, preferHTTPS bool) string {
	remoteURL = strings.TrimSpace(remoteURL)
	if !preferHTTPS || slug == "" {
		return remoteURL
	}

	if !isSSHRemote(remoteURL) {
		return remoteURL
	}

	// Only rewrite when the slug names a host: `github.com/owner/repo`
	// converts cleanly, `some-local-path` does not.
	host, _, found := strings.Cut(slug, "/")
	if !found || !strings.Contains(host, ".") {
		return remoteURL
	}

	return "https://" + slug + ".git"
}

// isSSHRemote reports whether a remote would need an ssh key.
func isSSHRemote(remoteURL string) bool {
	return strings.HasPrefix(remoteURL, "ssh://") ||
		(strings.Contains(remoteURL, "@") && strings.Contains(remoteURL, ":") && !strings.Contains(remoteURL, "://"))
}

// git runs a git command, returning its combined output.
func (w *Worker) git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// Never stop for a passphrase or a host key prompt; a stuck agent
	// is worse than a failed job.
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	sshCommand := w.installSSHKeys(ctx)
	if sshCommand == "" {
		sshCommand = "ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new"
	}
	cmd.Env = append(env, "GIT_SSH_COMMAND="+sshCommand)

	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
