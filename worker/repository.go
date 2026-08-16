package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/runner"
)

// Workspace is a checkout prepared for one job.
type Workspace struct {
	// Root is the work tree the job runs under.
	Root string

	// Dir is the directory the command runs in: Root joined with the
	// job's working directory.
	Dir string

	// Checkout is what the work tree ended up at.
	Checkout Checkout

	// Artefacts is the directory the job copies files into to have them
	// kept. It sits beside the work tree rather than inside it, so
	// staging a file does not dirty the checkout the job is testing.
	Artefacts string

	// Env is what the workspace contributes to the job's environment:
	// the paths it laid out and the checkout it produced. prepare fills
	// it, and a caller holding the workspace may add to it before the
	// command starts. The values here win over the ones the agent
	// derives from the job.
	Env runner.Env
}

// Checkout records what the agent actually put in the work tree.
//
// Ref is the effective ref: the one the job named, or the default branch
// resolved for a job that named nothing. CommitSHA is what that ref
// pointed at when the job ran — a tag moves, so a run recorded as
// "v1.2.3" alone cannot be reproduced later.
type Checkout struct {
	Ref       string
	CommitSHA string

	// Branch is set when Ref named a branch and empty for a tag or a
	// bare commit. A job reads it as ATKINS_BRANCH.
	Branch string
}

// prepare produces a checkout for a job.
//
// Clones are cached per repository under <DataDir>/repos, so the second
// job for a repository fetches instead of downloading it again. Each
// job then gets its own work tree built from that cache, which keeps
// concurrent jobs from fighting over one checkout.
func (w *Worker) prepare(ctx context.Context, job *jobContext) (*Workspace, error) {
	cache, err := w.cacheRepository(ctx, job)
	if err != nil {
		return nil, err
	}

	checkout, err := w.resolve(ctx, cache, job)
	if err != nil {
		return nil, err
	}

	root := filepath.Join(w.opts.DataDir, "work", job.Job.ID)
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}

	if err := w.workTree(ctx, cache, root, checkout.CommitSHA, cloneDepth(job.Job.CloneDepth)); err != nil {
		return nil, err
	}

	dir := root
	if relative := CleanWorkingDirectory(job.Job.WorkingDirectory); relative != "" {
		dir = filepath.Join(root, filepath.FromSlash(relative))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("working directory %q does not exist in the repository", relative)
		}
	}

	// The output directory exists before the command starts, so a
	// pipeline can copy into it without first testing whether it is
	// there.
	artefacts := root + artefactSuffix
	if err := os.MkdirAll(artefacts, 0o755); err != nil {
		return nil, fmt.Errorf("create artefact directory: %w", err)
	}

	workspace := &Workspace{Root: root, Dir: dir, Checkout: *checkout, Artefacts: artefacts}
	workspace.Env = workspace.environment()

	return workspace, nil
}

// environment is what the workspace publishes to the job command.
//
// ATKINS_REVISION keeps the name it has always had and holds the same
// resolved sha as ATKINS_COMMIT_SHA, so a pipeline reading either can
// pin an artefact to a commit even when the job named a branch or a
// moving tag. The checkout reported here is the one the agent produced
// rather than the one the job asked for.
func (w *Workspace) environment() runner.Env {
	env := runner.Env{
		"ATKINS_WORKSPACE":  w.Root,
		client.EnvArtefacts: w.Artefacts,
	}

	if w.Checkout.Ref != "" {
		env["ATKINS_REF"] = w.Checkout.Ref
	}
	if sha := w.Checkout.CommitSHA; sha != "" {
		env["ATKINS_COMMIT_SHA"] = sha
		env["ATKINS_REVISION"] = sha
	}
	if w.Checkout.Branch != "" {
		env["ATKINS_BRANCH"] = w.Checkout.Branch
	}

	return env
}

// artefactSuffix names the output directory beside a job's work tree,
// so one cleanup pass over <data>/work removes both.
const artefactSuffix = ".artefacts"

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
//
// The mirror is always complete, whatever depth a job asks for. It is
// the agent's copy of the remote rather than any one job's checkout: a
// shallow cache would have to be deepened the first time a job named an
// older tag, and deepening repeatedly costs more than never having
// thrown the objects away.
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

// resolve turns the ref a job named into a commit.
//
// Resolution happens against the cache, where refs carry the names the
// remote gives them, and it happens exactly once: the work tree is then
// built around the sha, so a tag that moves between resolving it and
// checking it out cannot split one job across two commits.
//
// A named ref that does not resolve is a failure, not an invitation to
// build something else. Falling through to the default branch is how a
// typo in a tag name turns into a green run of the wrong code.
func (w *Worker) resolve(ctx context.Context, cache string, job *jobContext) (*Checkout, error) {
	requested := strings.TrimSpace(job.Job.Ref)
	if requested == "" {
		return w.resolveDefault(ctx, cache, job)
	}

	for _, candidate := range refCandidates(requested) {
		sha := w.revision(ctx, cache, candidate)
		if sha == "" {
			continue
		}
		return &Checkout{Ref: requested, CommitSHA: sha, Branch: branchName(candidate)}, nil
	}

	// Naming the ref matters. "checkout failed" for a tag nobody pushed
	// sends whoever reads it looking at the agent instead of at the tag.
	return nil, fmt.Errorf("ref %q not found in %s", requested, job.Repository.Slug)
}

// resolveDefault picks what a job that named no ref should build: the
// repository's default branch, or whatever the mirror's HEAD points at
// when the server never learned one.
func (w *Worker) resolveDefault(ctx context.Context, cache string, job *jobContext) (*Checkout, error) {
	candidates := []string{}
	if branch := strings.TrimSpace(job.Repository.DefaultBranch); branch != "" {
		candidates = append(candidates, "refs/heads/"+branch)
	}
	if head, err := w.gitOutput(ctx, cache, "symbolic-ref", "--quiet", "HEAD"); err == nil && head != "" {
		candidates = append(candidates, head)
	}

	for _, candidate := range candidates {
		sha := w.revision(ctx, cache, candidate)
		if sha == "" {
			continue
		}

		branch := branchName(candidate)
		ref := branch
		if ref == "" {
			ref = candidate
		}
		return &Checkout{Ref: ref, CommitSHA: sha, Branch: branch}, nil
	}

	return nil, fmt.Errorf("%s has no default branch to check out", job.Repository.Slug)
}

// revision resolves one candidate to a commit sha, or "" when the cache
// has no such ref.
func (w *Worker) revision(ctx context.Context, dir, candidate string) string {
	sha, err := w.gitOutput(ctx, dir, "rev-parse", "--verify", "--quiet", candidate+"^{commit}")
	if err != nil {
		return ""
	}
	return sha
}

// refCandidates expands a ref into what it could name, in git's own
// order: a fully qualified ref as given, then a tag, then a branch, then
// anything else rev-parse understands — which is what lets a bare commit
// sha work without being declared as one.
func refCandidates(ref string) []string {
	if strings.HasPrefix(ref, "refs/") {
		return []string{ref}
	}
	return []string{"refs/tags/" + ref, "refs/heads/" + ref, ref}
}

// branchName returns the branch a candidate names, or "".
func branchName(candidate string) string {
	if branch, found := strings.CutPrefix(candidate, "refs/heads/"); found {
		return branch
	}
	return ""
}

// cloneDepth clamps a requested depth. A negative depth is a bad payload
// rather than an instruction, and 0 means the whole history.
func cloneDepth(depth int64) int64 {
	if depth < 0 {
		return 0
	}
	return depth
}

// workTree builds the job's work tree from the cache, at one commit.
//
// Depth belongs here and never on the cache. The cache is shared by
// every job for a repository, so one job asking for a single commit must
// not leave the next one refetching history the agent had already paid
// for; the work tree is thrown away when the job ends, so limiting it
// costs nothing later.
func (w *Worker) workTree(ctx context.Context, cache, root, sha string, depth int64) error {
	if depth > 0 {
		return w.shallowWorkTree(ctx, cache, root, sha, depth)
	}
	return w.fullWorkTree(ctx, cache, root, sha)
}

// fullWorkTree clones the whole cache into the work tree.
func (w *Worker) fullWorkTree(ctx context.Context, cache, root, sha string) error {
	// A local clone of the mirror is cheap: git hardlinks the objects.
	if out, err := w.git(ctx, "", "clone", "--no-checkout", cache, root); err != nil {
		return fmt.Errorf("clone work tree: %w\n%s", err, out)
	}

	// A clone takes branches and tags. A commit the mirror holds under
	// anything else — refs/pull/*, say — is not in the clone, so fetch it
	// rather than fail on a ref that demonstrably resolved a moment ago.
	if w.revision(ctx, root, sha) == "" {
		if out, err := w.git(ctx, root, "fetch", "--quiet", "--no-tags", "origin", sha); err != nil {
			return fmt.Errorf("fetch %s: %w\n%s", sha, err, out)
		}
	}

	return w.checkout(ctx, root, sha)
}

// shallowWorkTree builds a work tree holding depth commits of history.
//
// `git clone` ignores --depth for a local path — it hardlinks the whole
// object store instead — so the shallow work tree is assembled by hand:
// an empty repository, then a depth-limited fetch of the one commit that
// was already resolved.
func (w *Worker) shallowWorkTree(ctx context.Context, cache, root, sha string, depth int64) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create work tree: %w", err)
	}
	if out, err := w.git(ctx, root, "init", "--quiet"); err != nil {
		return fmt.Errorf("init work tree: %w\n%s", err, out)
	}
	if out, err := w.git(ctx, root, "remote", "add", "origin", cache); err != nil {
		return fmt.Errorf("add cache remote: %w\n%s", err, out)
	}

	// allowAnySHA1InWant lets the fetch name the commit rather than the
	// ref it came from: the ref was resolved against this cache already,
	// and resolving it a second time is a chance for it to have moved.
	out, err := w.git(ctx, root,
		"-c", "uploadpack.allowAnySHA1InWant=true",
		"fetch", "--quiet", "--depth", strconv.FormatInt(depth, 10), "origin", sha)
	if err != nil {
		return fmt.Errorf("fetch %s at depth %d: %w\n%s", sha, depth, err, out)
	}

	return w.checkout(ctx, root, sha)
}

// checkout detaches the work tree at a commit.
//
// Detaching is deliberate: a job builds one commit, and leaving a branch
// checked out would suggest the job has somewhere to push it back to.
func (w *Worker) checkout(ctx context.Context, root, sha string) error {
	if out, err := w.git(ctx, root, "checkout", "--force", "--detach", sha); err != nil {
		return fmt.Errorf("checkout %s: %w\n%s", sha, err, out)
	}
	return nil
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

// git runs a git command, returning its combined output. It is for the
// commands whose output is only ever read by a human diagnosing a
// failure.
func (w *Worker) git(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := w.gitCommand(ctx, dir, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// gitOutput runs a git command and returns its standard output alone.
//
// Combined output would fold a warning on stderr into a value the agent
// is about to treat as a commit sha.
func (w *Worker) gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := w.gitCommand(ctx, dir, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// gitCommand prepares a git invocation with the agent's environment.
func (w *Worker) gitCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
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

	return cmd
}
