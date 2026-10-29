// Package config owns the atkins configuration document, .atkins/config.yml.
//
// Configuration comes in layers, each overriding the last: the embedded
// default (RuntimeDefaultConfig), the user document at
// $HOME/.atkins/config.yml, the project document found by walking up from
// the working directory, and the ATKINS_* environment overlay. Load
// assembles them; LoadFile reads one document over the default, and skips
// the user layer and the environment, for the configuration menu to edit.
//
// The embedded default holds a complete and valid set of values. An
// unconfigured install runs on those values. Every field a document
// leaves empty keeps the default, because a document unmarshals over the
// struct holding the layers below it. The menu writes the embedded
// document, comments and all, when a project has none yet. The file a
// user first opens explains itself.
//
// The environment overlay serves containers and CI. `env` struct tags
// drive it, and a field added to the document gains an override with no
// further code. It applies a variable only when the variable holds a
// value: an empty ATKINS_* variable means "not configured here". A
// container can set one field and keep the rest of the document. It skips
// a value that does not parse and carries on, and a container with a typo
// in an optional setting still starts.
//
// Validation fills what remains empty and checks the version. The
// document is two fields wide: the run that opens the database reports a
// path that does not open, and recording is a boolean. A document from a
// newer build may hold fields this build ignores, and a newer version
// fails validation.
package config
