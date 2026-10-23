// Package config owns the atkins configuration document.
//
// Configuration has two layers. The document (.atkins/config.yml,
// falling back to the embedded RuntimeDefaultConfig) is the source of
// truth. Environment variables are an overlay on top of it, for
// containers and CI, and only where they are actually set: an unset or
// empty ATKINS_* variable leaves the configured value alone.
//
// Load checks every field, and an empty one takes the default from the
// embedded document. A half-written configuration fails at start-up
// with a message naming the field, long before the first request.
package config

// Version is the document version this build understands.
const Version = 1

// Config is the .atkins/config.yml document.
type Config struct {
	// Version identifies the document format.
	Version int `yaml:"version" json:"version"`

	// Storage is what a run records and where.
	Storage StorageConfig `yaml:"storage" json:"storage"`
}

// StorageConfig is the event database: the file every run appends a row
// per job and per command to, and what is kept in it.
type StorageConfig struct {
	// Database is the file to record into. Empty discovers it: the
	// atkins.db of the nearest .atkins folder, and
	// $HOME/.atkins/atkins.db for a directory that belongs to no
	// project.
	Database string `yaml:"database" json:"database" env:"ATKINS_DB"`

	// Record writes a row per job and per command of every run. Off
	// records nothing, the same as `--local`, and leaves the database
	// alone.
	//
	// How long a run is kept is not a setting: storage.Retention is 31
	// days on every machine, applied by the run that records.
	Record bool `yaml:"record" json:"record" env:"ATKINS_RECORD"`
}
