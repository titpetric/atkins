# Package ./server/schema

```go
import (
	"github.com/titpetric/atkins/server/schema"
}
```

Package schema holds the SQL migrations for the atkins CI/CD server.

The migrations are the source of truth for the data model. One file
owns one table, and files are append-only once applied anywhere: a
later change belongs in a new *.up.sql file.

Run `atkins "$HOME/.atkins/skills/schema.yml" -w ./server migrate` to
regenerate ../model and ./docs after changing any migration.

## Function symbols

- `func Migrations () fs.FS`

### Migrations

Migrations returns the embedded migrations filesystem.

```go
func Migrations() fs.FS
```
