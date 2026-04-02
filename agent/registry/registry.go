package registry

import "strings"

// Registry is a generic container for registered items.
type Registry[T any] struct {
	commands map[string]T
	ordered  []string
}

// New creates a new registry.
func New[T any]() *Registry[T] {
	return &Registry[T]{commands: make(map[string]T)}
}

// Register adds an item to the registry.
func (r *Registry[T]) Register(name string, aliases []string, cmd T) {
	name = strings.ToLower(name)
	r.commands[name] = cmd
	r.ordered = append(r.ordered, name)
	for _, alias := range aliases {
		r.commands[strings.ToLower(alias)] = cmd
	}
}

// Get retrieves an item by name or alias.
func (r *Registry[T]) Get(name string) (T, bool) {
	cmd, ok := r.commands[strings.ToLower(name)]
	return cmd, ok
}

// Names returns the ordered list of registered names.
func (r *Registry[T]) Names() []string {
	return r.ordered
}

// GetByName returns the item registered under the given name (not alias).
func (r *Registry[T]) GetByName(name string) (T, bool) {
	for _, n := range r.ordered {
		if n == name {
			return r.commands[n], true
		}
	}
	var zero T
	return zero, false
}
