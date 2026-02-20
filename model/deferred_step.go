package model

// DeferredStep represents a deferred step wrapper.
type DeferredStep struct {
	Defer *Step `yaml:"defer,omitempty"`
}
