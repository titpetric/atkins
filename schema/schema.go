// Package schema holds the SQL migrations of the atkins event database.
//
// The migrations are the source of truth for the data model. One file
// owns one table, and a file is append-only once it has been applied
// anywhere: a later change to a table belongs in a new *.up.sql file,
// because mig records progress per statement index.
//
// Run `atkins migrate` after changing a migration. The job comes from
// the schema skill and regenerates ../model and ./docs.
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
