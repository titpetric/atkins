// Package storage owns the SQL for the atkins CI/CD server.
//
// Queries run through github.com/titpetric/pdo, whose generic methods
// (Get[T], Select[T], Insert, Update) scan straight into the mig-generated
// model types. A *pdo.PDO is request-scoped and not safe for concurrent
// use, so every method allocates one over the shared *sqlx.DB pool.
//
// Callers outside this package should not issue SQL; add a method here
// instead so transactions and scoping stay in one place.
package storage

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/pdo"
	"github.com/titpetric/platform"
)

// ConnectionName is the platform database connection the server prefers.
// Configure either of:
//
//	PLATFORM_DB_ATKINS="sqlite://file:atkins.db"
//	PLATFORM_DB_DEFAULT="sqlite://file:atkins.db"
//
// The named connection wins when both are set. With neither, the
// platform default is an in-memory sqlite database, which is useful for
// tests and useless for a server: the server command sets
// PLATFORM_DB_DEFAULT to a file when the environment doesn't.
const ConnectionName = "atkins"

// DB returns the shared connection pool for the atkins server.
//
// An empty name selects ConnectionName. Naming it explicitly lets one
// process host more than one atkins database, which is what the module
// tests use to keep their fixtures apart: the platform caches pools by
// connection name for the life of the process.
func DB(ctx context.Context, name string) (*sqlx.DB, error) {
	if name == "" {
		name = ConnectionName
	}
	return platform.Database.Connect(ctx, name, "default")
}

// client returns a request-scoped pdo client over the shared pool.
func client(db *sqlx.DB) *pdo.PDO {
	return pdo.New(db)
}
