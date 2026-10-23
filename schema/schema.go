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
