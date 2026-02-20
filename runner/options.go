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
		DefaultTimeout: 300 * time.Second, // 5 minutes default
	}
}
