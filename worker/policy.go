package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/server/model"
)

// policy caches the repository policy the agent enforces.
//
// The server refuses a disallowed dispatch, so this is a second gate on
// the same decision. It matters because a job outlives the rule that
// admitted it: a repository removed from the allowlist must not be
// built by a job that was queued while it was still listed.
type policy struct {
	mu        sync.RWMutex
	value     client.PolicyResponse
	fetchedAt time.Time
	loaded    bool
}

// policyTTL is how long a fetched policy is trusted before refreshing.
// Short, because it is the thing standing between a revoked rule and a
// clone that shouldn't happen.
const policyTTL = 30 * time.Second

// refresh fetches the policy when the cached copy is stale.
func (w *Worker) refresh(ctx context.Context) error {
	w.policy.mu.RLock()
	fresh := w.policy.loaded && time.Since(w.policy.fetchedAt) < policyTTL
	w.policy.mu.RUnlock()

	if fresh {
		return nil
	}

	value, err := w.client.Policy(ctx)
	if err != nil {
		return err
	}

	w.policy.mu.Lock()
	w.policy.value = *value
	w.policy.fetchedAt = time.Now()
	w.policy.loaded = true
	w.policy.mu.Unlock()

	return nil
}

// allowed reports whether the agent may build a repository.
//
// A policy it could not fetch is treated as a refusal. An agent that
// cannot tell what it is allowed to run should run nothing; the
// alternative is that a server outage turns an allowlist into an
// open door.
func (w *Worker) allowed(ctx context.Context, slug string) (bool, error) {
	if err := w.refresh(ctx); err != nil {
		return false, fmt.Errorf("could not read the repository policy: %w", err)
	}

	w.policy.mu.RLock()
	defer w.policy.mu.RUnlock()

	if w.policy.value.Policy != model.PolicyAllowlist {
		return true, nil
	}

	for _, pattern := range w.policy.value.Patterns {
		if model.MatchRepository(pattern, slug) {
			return true, nil
		}
	}

	return false, nil
}

// logPolicy reports the policy in force at start-up, so an operator can
// see from the logs whether the agent is gated.
func (w *Worker) logPolicy(ctx context.Context) {
	if err := w.refresh(ctx); err != nil {
		log.Printf("[agent] could not read the repository policy: %v", err)
		return
	}

	w.policy.mu.RLock()
	defer w.policy.mu.RUnlock()

	if w.policy.value.Policy == model.PolicyAllowlist {
		log.Printf("[agent] repository policy: allowlist (%d patterns: %v)",
			len(w.policy.value.Patterns), w.policy.value.Patterns)
		return
	}

	log.Printf("[agent] repository policy: open")
}
