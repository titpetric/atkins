package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/server/model"
)

// Limits on one job's collection. The server enforces its own, from the
// setting registry; these keep a runaway pipeline from turning into
// thousands of pointless requests before the server says no.
const (
	// maxCollected bounds how many files one job may offer.
	maxCollected = 200

	// uploadTimeout bounds a single artefact transfer.
	uploadTimeout = 5 * time.Minute
)

// candidate is one file to upload.
type candidate struct {
	// path is the name the artefact is stored under, relative to the
	// directory the job ran in.
	path string

	// source is where the file actually is on this agent.
	source string
}

// collect uploads what the job produced and reports how many artefacts
// the server accepted.
//
// It runs whether the command passed or failed, on purpose: the files
// produced by a failure — a coverage report, a scan that found
// something, a core dump — are usually the ones worth having.
func (w *Worker) collect(ctx context.Context, job *jobContext, workspace *Workspace) int {
	candidates := w.candidates(job, workspace)
	if len(candidates) == 0 {
		return 0
	}

	// The job's own deadline has usually just expired when it timed
	// out, and the artefacts of a timeout are worth as much as any
	// others.
	ctx = context.WithoutCancel(ctx)

	uploaded := 0
	for _, item := range candidates {
		if err := w.upload(ctx, job.Job.ID, item); err != nil {
			w.appendLog(ctx, job.Job.ID, client.StreamError,
				fmt.Sprintf("artefact %s was not stored: %v\n", item.path, err))
			log.Printf("[agent] job %s: artefact %s failed: %v", job.Job.ID, item.path, err)
			continue
		}
		uploaded++
	}

	if uploaded > 0 {
		log.Printf("[agent] job %s: stored %d artefact(s)", job.Job.ID, uploaded)
	}

	return uploaded
}

// upload streams one file to the server.
func (w *Worker) upload(ctx context.Context, jobID string, item candidate) error {
	checksum, err := checksumFile(item.source)
	if err != nil {
		return err
	}

	file, err := os.Open(item.source)
	if err != nil {
		return err
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	_, err = w.client.UploadArtefact(ctx, jobID, client.ArtefactUpload{
		Path:        item.path,
		ContentType: contentType(item.path),
		Checksum:    checksum,
		Content:     file,
	})
	return err
}

// candidates gathers the files a job asked to keep, from the two places
// a job can put them.
func (w *Worker) candidates(job *jobContext, workspace *Workspace) []candidate {
	found := map[string]string{}

	// The output directory: everything a job copied into
	// $ATKINS_ARTEFACTS, named by its path within that directory.
	walkTree(workspace.Artefacts, nil, found)

	// The declared globs, matched against the checkout the command ran
	// in. A job that already writes coverage.json where it writes it
	// can be collected without editing the pipeline.
	if patterns := model.ArtefactPatterns(job.Job.ArtefactPaths); len(patterns) > 0 {
		walkTree(workspace.Dir, patterns, found)
	}

	paths := make([]string, 0, len(found))
	for name := range found {
		paths = append(paths, name)
	}
	sort.Strings(paths)

	if len(paths) > maxCollected {
		log.Printf("[agent] job %s: %d files matched, uploading the first %d",
			job.Job.ID, len(paths), maxCollected)
		paths = paths[:maxCollected]
	}

	candidates := make([]candidate, 0, len(paths))
	for _, name := range paths {
		candidates = append(candidates, candidate{path: name, source: found[name]})
	}
	return candidates
}

// walkTree collects files under root into found, keyed by their path
// relative to root. A nil patterns list takes everything.
//
// Collection walks the tree rather than expanding the patterns into
// paths. That is the difference between a pattern that can only select
// among files that are already inside the checkout, and one that gets
// handed to the filesystem and can name `../../etc/passwd`. WalkDir
// does not follow symlinks either, so a link planted in a repository
// cannot widen the walk to somewhere else on the agent.
func walkTree(root string, patterns []string, found map[string]string) {
	if strings.TrimSpace(root) == "" {
		return
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return
	}

	_ = filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable directory is not worth failing the job for;
			// the rest of the tree is still collectable.
			return nil
		}

		if entry.IsDir() {
			// Never the repository's own object store: it is large, it
			// is not an output, and nobody asked for it.
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only regular files: a symlink, a socket or a device is not an
		// artefact, and following one is how a walk leaves its tree.
		if !entry.Type().IsRegular() {
			return nil
		}

		relative, err := filepath.Rel(root, name)
		if err != nil {
			return nil
		}

		stored := model.ArtefactPath(filepath.ToSlash(relative))
		if stored == "" {
			return nil
		}
		if !matchesAny(patterns, stored) {
			return nil
		}

		// First writer wins, so an explicitly staged file is not
		// replaced by a glob match with the same name.
		if _, taken := found[stored]; !taken {
			found[stored] = name
		}
		return nil
	})
}

// matchesAny reports whether a path is wanted. No patterns means the
// whole tree was asked for.
func matchesAny(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if model.MatchArtefactPath(pattern, value) {
			return true
		}
	}
	return false
}

// checksumFile hashes a file so the server can tell a complete upload
// from a truncated one.
func checksumFile(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// contentType guesses a media type from the file extension. The server
// falls back to application/octet-stream for anything unrecognized, so
// a guess that fails costs nothing.
func contentType(name string) string {
	return mime.TypeByExtension(path.Ext(name))
}
