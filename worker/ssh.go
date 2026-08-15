package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/titpetric/atkins/client"
)

// sshKeys caches the deploy keys installed on disk.
type sshKeys struct {
	mu sync.RWMutex

	// command is the GIT_SSH_COMMAND that offers the installed keys.
	command string

	fetchedAt time.Time
	loaded    bool
}

// sshKeyTTL is how long installed keys are trusted before refetching,
// so a revoked key stops working within the minute.
const sshKeyTTL = time.Minute

// installSSHKeys writes the server's deploy keys into the agent's data
// directory and builds the GIT_SSH_COMMAND that offers them.
//
// Keys live under <DataDir>/ssh at 0600 in a 0700 directory. They are
// written to disk because that is the only interface git has for
// identities; nothing else on the agent should be able to read them.
func (w *Worker) installSSHKeys(ctx context.Context) string {
	w.ssh.mu.RLock()
	fresh := w.ssh.loaded && time.Since(w.ssh.fetchedAt) < sshKeyTTL
	command := w.ssh.command
	w.ssh.mu.RUnlock()

	if fresh {
		return command
	}

	keys, err := w.client.SSHKeys(ctx)
	if err != nil {
		// Without keys, public https clones still work. Say so once
		// rather than failing every job.
		log.Printf("[agent] could not fetch ssh keys: %v", err)
		return command
	}

	command, err = w.writeSSHKeys(keys)
	if err != nil {
		log.Printf("[agent] could not install ssh keys: %v", err)
		return ""
	}

	w.ssh.mu.Lock()
	w.ssh.command = command
	w.ssh.fetchedAt = time.Now()
	w.ssh.loaded = true
	w.ssh.mu.Unlock()

	if len(keys) > 0 {
		log.Printf("[agent] installed %d ssh key(s)", len(keys))
	}

	return command
}

// writeSSHKeys puts the keys on disk and returns the ssh command.
func (w *Worker) writeSSHKeys(keys []client.AgentSSHKey) (string, error) {
	dir := filepath.Join(w.opts.DataDir, "ssh")

	// Rewrite from scratch so a key deleted on the server disappears
	// here too, rather than lingering as a file git can still use.
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if len(keys) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	// Deterministic order: git tries identities in the order given,
	// and a stable order makes a failure reproducible.
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name < keys[j].Name })

	var identities []string
	var knownHosts []string

	for _, key := range keys {
		path := filepath.Join(dir, sanitizeName(key.Name))
		if err := os.WriteFile(path, []byte(ensureTrailingNewline(key.PrivateKey)), 0o600); err != nil {
			return "", fmt.Errorf("write ssh key %s: %w", key.Name, err)
		}
		identities = append(identities, "-i "+path)

		if hosts := strings.TrimSpace(key.KnownHosts); hosts != "" {
			knownHosts = append(knownHosts, hosts)
		}
	}

	options := []string{
		"ssh",
		"-o BatchMode=yes",
		// Only the listed identities: without this ssh also offers
		// whatever an agent socket holds, and the first refusal can
		// end the handshake.
		"-o IdentitiesOnly=yes",
	}

	if len(knownHosts) > 0 {
		path := filepath.Join(dir, "known_hosts")
		content := ensureTrailingNewline(strings.Join(knownHosts, "\n"))
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return "", fmt.Errorf("write known_hosts: %w", err)
		}
		options = append(options,
			"-o UserKnownHostsFile="+path,
			"-o StrictHostKeyChecking=yes",
		)
	} else {
		// Nothing pinned: trust on first use rather than prompting,
		// which would hang the job.
		options = append(options, "-o StrictHostKeyChecking=accept-new")
	}

	return strings.Join(append(options, identities...), " "), nil
}

// sanitizeName turns a key name into a safe file name.
func sanitizeName(name string) string {
	replaced := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.':
			return r
		default:
			return '-'
		}
	}, name)

	replaced = strings.Trim(replaced, ".-")
	if replaced == "" {
		return "key"
	}
	return replaced
}

// ensureTrailingNewline is required by ssh for key files.
func ensureTrailingNewline(value string) string {
	value = strings.TrimRight(value, "\n") + "\n"
	return value
}
