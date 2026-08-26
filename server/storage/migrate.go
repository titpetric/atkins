package storage

import (
	"context"
	"io/fs"

	"github.com/go-bridget/mig/migrate"
	"github.com/jmoiron/sqlx"
)

// Project is the migration project name recorded in the migrations table.
const Project = "atkins"

// Migrate applies SQL migrations from the given filesystem to the database.
// Files are selected by mig's default "*.up.sql" pattern; a filesystem with
// none of them is an error.
func Migrate(ctx context.Context, db *sqlx.DB, schema fs.FS) error {
	m, err := migrate.NewManager(db, schema, Project)
	if err != nil {
		return err
	}

	_, err = m.Apply(ctx)
	return err
}
