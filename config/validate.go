package config

import (
	"fmt"
)

// Validate fills empty fields from the embedded defaults and checks what
// remains.
//
// The document is two fields wide, and neither can be wrong in a way
// worth refusing to start over: a database path that does not open is
// reported by the run that tried, and recording is a boolean. What is
// left to check is the version, because a document from a newer build
// may hold fields this one would silently ignore.
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
