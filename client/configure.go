package client

import (
	"sync"
	"time"

	"github.com/titpetric/atkins/config"
)

// settings holds the client configuration for this process.
//
// The package reads configuration rather than the environment so that
// .atkins/config.yml is the source of truth and ATKINS_* is only the
// overlay the config package already applied. Commands that never call
// Configure still work: the zero value behaves as an unconfigured
// machine, which is the historical behaviour.
var settings struct {
	mu    sync.RWMutex
	value config.ClientConfig
	set   bool
}

// Configure installs the resolved client configuration.
func Configure(value config.ClientConfig) {
	settings.mu.Lock()
	defer settings.mu.Unlock()

	settings.value = value
	settings.set = true
}

// Settings returns the configuration in force.
func Settings() config.ClientConfig {
	settings.mu.RLock()
	defer settings.mu.RUnlock()

	if !settings.set {
		return config.ClientConfig{Record: true, Timeout: DefaultTimeout}
	}
	return settings.value
}

// configuredServer returns the server to talk to, preferring an
// explicit argument over configuration.
func configuredServer(server string) string {
	if server != "" {
		return NormalizeServer(server)
	}
	return NormalizeServer(Settings().Server)
}

// configuredTimeout returns the API call timeout.
func configuredTimeout() time.Duration {
	if timeout := Settings().Timeout; timeout > 0 {
		return timeout
	}
	return DefaultTimeout
}
