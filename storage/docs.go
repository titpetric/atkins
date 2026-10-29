// Package storage owns the atkins event database.
//
// Every run leaves two kinds of row behind: a `run` row per node of the
// tree atkins walked, carrying the time that node took with everything
// under it, and a `command` row per command a step executed. The first
// answers "how long does the release job take", the second "how much of
// my day goes to go test".
//
// # Files
//
// A run records to two databases, and Discover names them. The
// project's, atkins.db in the nearest .atkins folder, says what this
// checkout costs. The workspace's, $HOME/.atkins/atkins.db or the first
// file the database setting or ATKINS_DB names, says what every checkout
// on the machine costs between them, a question no single project can
// answer. A directory with no .atkins folder of its own, or one whose
// nearest folder belongs to the workspace, records to the workspace's
// alone, once.
//
// The other files the setting names, a list in the form of PATH, are
// mirrors of the workspace's: each takes every row the workspace's takes,
// and no report reads one. A runner that sets a $HOME per checkout
// names the checkout's file first and its own second, and keeps one
// history across them. atkins reports a mirror that fails to open once,
// and the rest take the row.
//
// Both files hold the same rows. A report of the workspace answers the
// questions of a project report: which command, how often, how long. A
// database that opens stays in use when the other fails: atkins keeps
// the half of the history it can reach, and the report names the file
// each half comes from. A failed write to one database leaves the other one's row in
// place, and the caller sees the first error.
//
// # One surface
//
// One type, Storage, owns a database. Every method takes a context and
// one request value and returns one result and an error. A reader, the
// --stats report today and task-ui later, has one surface to learn.
// Queries go through github.com/titpetric/pdo, whose generic methods
// scan straight into the mig generated types in ../model. A *pdo.PDO
// serves one request at a time, and every method allocates one over the
// shared pool.
//
// A file opens with a busy timeout and the write-ahead log: two atkins
// runs in one checkout write at the same moment, and the second waits
// for the first to finish. The driver writes a time in a layout strftime
// reads, and the usage buckets group by it; time.Time's own String form
// leaves every bucket empty.
//
// # Recording
//
// Record writes the rows of one run in one transaction. A report reads
// a run as a whole, the job's time against its steps', and a half
// written run would pass for a complete one.
//
// A job row takes the canonical name out of the node id, so `jobs.release`
// groups as release from one month to the next while the author edits
// its description. A step row keeps the command it ran, the name a reader
// of one run looks for. Durations round to the millisecond: truncation
// drops half a millisecond per row, and a report summing thousands of
// rows comes up seconds short.
//
// # Retention
//
// Retention, thirty-one days, bounds the history. A month of work fits
// in it, this week compares against the last, and a checkout nobody
// prunes stops growing. Every machine uses the same period. The Recorder purges after each write: the command that
// records has the database open anyway, and atkins stays a command with
// no service beside it. Purge deletes in bounded batches and leaves the
// rest for the next pass, because one DELETE over a year of commands
// holds the write lock while every other atkins run on the machine
// waits on it.
//
// # Command keys
//
// Normalize reduces a command to the form a row stores: the tool and its
// subcommands, then the flags with their values removed, sorted.
//
//	go test -v ./...                      go test -v
//	go test ./... -v                      go test -v
//	go test -cover -coverprofile=c.out    go test -cover -coverprofile
//	go test --cpuprofile cpu.out          go test --cpuprofile
//	CGO_ENABLED=0 go build -o bin/x .     go build -o
//	sleep 1                               sleep
//	head -20 file.txt                     head
//	git rev-parse HEAD | cut -c1-8        git rev-parse HEAD
//
// Sorting makes `go test -v -cover` and `go test -cover -v` one row.
// Operands drop out, and a flag's argument drops with them with no table
// of the flags that take one: `--cpuprofile cpu.out` keeps the flag and
// the file reads as an operand. A flag made of digits goes too, so
// `head -20` and `head -50` count together.
//
// Key is the name the reports group by: the words of a command up to the
// first flag, at most three, none a path or a value. A word holding `/`,
// `.`, `=`, `$`, a quote or a number ends it.
//
//	go test -v ./...        go test
//	docker compose up -d    docker compose up
//	git clone --depth 1 x   git clone
//
// A normalized command has the same key as the command it came from. The
// reports group rows by key, and the stored form needs no migration. A shell
// builtin, `set -eu` at the top of a script among them, never reaches a
// row or a report.
package storage
