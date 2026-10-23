package config

import (
	_ "embed"
)

// RuntimeDefaultConfig is the default configuration document, the base
// layer under every other.
//
//go:embed config.yml
var RuntimeDefaultConfig []byte
