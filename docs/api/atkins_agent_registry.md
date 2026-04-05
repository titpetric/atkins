# Package ./agent/registry

```go
import (
	"github.com/titpetric/atkins/agent/registry"
}
```

## Types

```go
// Registry is a generic container for registered items.
type Registry[T any] struct {
	commands map[string]T
	ordered  []string
}
```

## Function symbols

- `func New () *Registry[T]`
- `func (*Registry[T]) Get (name string) (T, bool)`
- `func (*Registry[T]) GetByName (name string) (T, bool)`
- `func (*Registry[T]) Names () []string`
- `func (*Registry[T]) Register (name string, aliases []string, cmd T)`

### New

New creates a new registry.

```go
func New() *Registry[T]
```

### Get

Get retrieves an item by name or alias.

```go
func (*Registry[T]) Get(name string) (T, bool)
```

### GetByName

GetByName returns the item registered under the given name (not alias).

```go
func (*Registry[T]) GetByName(name string) (T, bool)
```

### Names

Names returns the ordered list of registered names.

```go
func (*Registry[T]) Names() []string
```

### Register

Register adds an item to the registry.

```go
func (*Registry[T]) Register(name string, aliases []string, cmd T)
```
