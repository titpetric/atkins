package storage

import (
	"time"
)

// CommandStat is one command key and what it has cost, a row of the
// command report.
type CommandStat struct {
	// Command is the key the rows group by, as Key names it: `go test`
	// counts every test run whatever its flags and packages.
	Command string

	// Tool is the first word of it.
	Tool string

	// Runs is how many times it ran.
	Runs int64

	// TotalMs is the time it has taken altogether, the order of the
	// report: a command that takes a second and runs four hundred times
	// costs more than a build that takes a minute.
	TotalMs int64

	// AvgMs and MaxMs are the average and the worst single run.
	AvgMs int64
	MaxMs int64

	// Failures is how many of the runs exited non-zero.
	Failures int64

	// LastAt is when it last ran.
	LastAt *time.Time
}
