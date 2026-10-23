# Package ./schema

```go
import (
	"github.com/titpetric/atkins/schema"
)
```

Package `schema` holds the SQL migrations of the atkins event database.

The migrations are the source of truth for the data model. One file
owns one table, and a file is append-only once any database runs it:
a later change to a table belongs in a new *.up.sql file, because mig
records progress per statement index.

Run `atkins migrate` after changing a migration. The job comes from
the schema skill and regenerates ../model and ./docs.

## Function symbols

- `func Migrations () fs.FS`

### `Migrations`

`Migrations` returns the embedded migrations filesystem.

```go
func Migrations() fs.FS
```
