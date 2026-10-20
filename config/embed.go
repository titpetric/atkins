package config

import (
	_ "embed"
)

// RuntimeDefaultConfig is the default configuration document, embedded
// so an unconfigured install still has a complete and valid set of
// values and no scatter of zero values.
//
// The configuration menu also writes it when a project has no
// .atkins/config.yml yet, comments and all, so the file a user first
// opens explains itself.
//
//go:embed config.yml
var RuntimeDefaultConfig []byte
