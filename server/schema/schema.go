// Package schema holds the SQL migrations for the atkins CI/CD server.
//
// The migrations are the source of truth for the data model. One file
// owns one table, and files are append-only once applied anywhere: a
// later change belongs in a new *.up.sql file.
//
// Run `atkins "$HOME/.atkins/skills/schema.yml" -w ./server migrate` to
// regenerate ../model and ./docs after changing any migration.
package schema

import (
	"embed"
	"io/fs"
)

//go:embed *.up.sql
var migrations embed.FS

// Migrations returns the embedded migrations filesystem.
func Migrations() fs.FS {
	return migrations
}
