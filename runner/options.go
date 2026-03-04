package runner

import (
	"time"
)

// Options provides configuration for the executor.
type Options struct {
	DefaultTimeout time.Duration
}

// DefaultOptions returns the default executor options.
func DefaultOptions() *Options {
	return &Options{
		DefaultTimeout: 900 * time.Second, // 15 minutes default
	}
}
