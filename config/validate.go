package config

import (
	"fmt"
)

// Validate fills empty fields from the embedded defaults, and returns an
// error for a version newer than this build.
func (c *Config) Validate() error {
	defaults, err := Default()
	if err != nil {
		return err
	}

	if c.Version == 0 {
		c.Version = defaults.Version
	}
	if c.Version > Version {
		return fmt.Errorf("config version %d is newer than this build understands (%d)", c.Version, Version)
	}

	return nil
}
