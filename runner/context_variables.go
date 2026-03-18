package runner

import (
	"fmt"
	"sync"

	"github.com/titpetric/atkins/model"
)

// Ensure ContextVariables implements model.VariableStorage
var _ model.VariableStorage = (*ContextVariables)(nil)

// promiseState tracks the resolution lifecycle of a variable.
type promiseState int

const (
	statePending   promiseState = iota // not yet resolved
	stateResolving                     // resolution in progress (cycle detection)
	stateResolved                      // resolution complete
)

// VarPromise represents a lazy variable that is evaluated on first access.
type VarPromise struct {
	value any
	raw   any
	err   error
	state promiseState
	mu    sync.Mutex
}

// newResolvedPromise creates a promise that is already resolved.
func newResolvedPromise(value any) *VarPromise {
	return &VarPromise{value: value, state: stateResolved}
}

// newPendingPromise creates a promise that will be lazily resolved.
func newPendingPromise(raw any) *VarPromise {
	return &VarPromise{raw: raw, state: statePending}
}

// ContextVariables provides thread-safe variable storage with Promise-based lazy evaluation.
type ContextVariables struct {
	promises map[string]*VarPromise
	resolver func(string) (string, error)
	mu       sync.Mutex
}

// NewContextVariables creates a ContextVariables from a map of evaluated values.
func NewContextVariables(values map[string]any) *ContextVariables {
	cv := &ContextVariables{
		promises: make(map[string]*VarPromise),
	}
	if values != nil {
		for k, v := range values {
			cv.promises[k] = newResolvedPromise(v)
		}
	}
	return cv
}

// NewContextVariablesWithResolver creates a ContextVariables with pending values
// that are evaluated lazily on first access via Get().
func NewContextVariablesWithResolver(pending map[string]any, resolver func(string) (string, error)) *ContextVariables {
	cv := &ContextVariables{
		promises: make(map[string]*VarPromise),
		resolver: resolver,
	}
	if pending != nil {
		for k, v := range pending {
			cv.promises[k] = newPendingPromise(v)
		}
	}
	return cv
}

// SetResolver updates the resolver function. Used when the resolver
// needs a reference to the ContextVariables itself (circular setup).
func (v *ContextVariables) SetResolver(resolver func(string) (string, error)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.resolver = resolver
}

// Get returns a variable value, evaluating lazily if pending.
func (v *ContextVariables) Get(key string) any {
	v.mu.Lock()
	promise, ok := v.promises[key]
	resolver := v.resolver
	v.mu.Unlock()

	if !ok {
		return nil
	}

	val, _ := v.resolve(promise, resolver)
	return val
}

// resolve evaluates a single promise, detecting cycles via the resolving state.
// The resolver may call back into Get() for other variables (dependency chain).
// If re-entry hits the same promise (cycle), it returns an error instead of deadlocking.
func (v *ContextVariables) resolve(p *VarPromise, resolver func(string) (string, error)) (any, error) {
	p.mu.Lock()

	switch p.state {
	case stateResolved:
		val, err := p.value, p.err
		p.mu.Unlock()
		return val, err

	case stateResolving:
		// Re-entrant call — this is a cycle.
		// Store the error on the promise so ResolveAll surfaces it.
		p.err = fmt.Errorf("circular variable dependency detected for %q", p.raw)
		p.value = p.raw
		p.state = stateResolved
		err := p.err
		p.mu.Unlock()
		return p.value, err

	default: // statePending
		p.state = stateResolving
		raw := p.raw
		p.mu.Unlock()

		// Resolve outside the lock — the resolver may call Get() on other vars
		var value any
		var err error

		if strVal, isStr := raw.(string); isStr && resolver != nil {
			resolved, resolveErr := resolver(strVal)
			if resolveErr != nil {
				value = strVal // On error, keep the raw string
				err = resolveErr
			} else {
				value = resolved
			}
		} else {
			value = raw
		}

		p.mu.Lock()
		if p.state == stateResolved && p.err != nil {
			// Cycle was detected during resolution — preserve the cycle error
			val, cycleErr := p.value, p.err
			p.mu.Unlock()
			return val, cycleErr
		}
		p.value = value
		p.err = err
		p.state = stateResolved
		p.mu.Unlock()

		return value, err
	}
}

// Set stores a value directly (already resolved).
func (v *ContextVariables) Set(key string, value any) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.promises[key] = newResolvedPromise(value)
}

// Clone creates a copy with the same resolver.
func (v *ContextVariables) Clone() model.VariableStorage {
	v.mu.Lock()
	defer v.mu.Unlock()

	clone := &ContextVariables{
		promises: make(map[string]*VarPromise, len(v.promises)),
		resolver: v.resolver,
	}
	for k, p := range v.promises {
		p.mu.Lock()
		if p.state == stateResolved {
			clone.promises[k] = newResolvedPromise(p.value)
		} else {
			clone.promises[k] = newPendingPromise(p.raw)
		}
		p.mu.Unlock()
	}
	return clone
}

// Walk iterates over all resolved values.
func (v *ContextVariables) Walk(fn func(key string, value any)) {
	v.mu.Lock()
	defer v.mu.Unlock()

	for k, p := range v.promises {
		p.mu.Lock()
		if p.state == stateResolved {
			fn(k, p.value)
		}
		p.mu.Unlock()
	}
}

// ResolveAll evaluates all pending promises and returns the first error encountered.
func (v *ContextVariables) ResolveAll() error {
	v.mu.Lock()
	keys := make([]string, 0, len(v.promises))
	for k := range v.promises {
		keys = append(keys, k)
	}
	resolver := v.resolver
	v.mu.Unlock()

	for _, key := range keys {
		v.mu.Lock()
		p, ok := v.promises[key]
		v.mu.Unlock()

		if !ok {
			continue
		}
		if _, err := v.resolve(p, resolver); err != nil {
			return err
		}
	}
	return nil
}
