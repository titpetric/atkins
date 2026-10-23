---
title: Statistics
subtitle: The event database and what it answers
layout: page
---

Every run records what it did in a SQLite file, and `atkins --stats` reports the commands by the time they have taken altogether. The question it answers is where a repository's day goes: not which command is slowest, but which one costs the most once its frequency is counted.

```bash
atkins --stats
```

The report has three parts: a column per day of the window, the tools ranked by the time they have taken, and the commands in a table.

```
/w/atkins/.atkins/atkins.db
31 days, 2026-09-22 to 2026-10-23, 30.5s across 624 runs

2.9s │    ▂     ▂▃     █▄▄         ▂
     │    █     ██    ▅███         █▃▄
     │    █     ██    ████     ▃▇  ████
     │    █     ██    ████     ██  ████
     │    █     ██    ████     ██  ████
     │    █     ██    ████     ██  ████
   0 └─────────────────────────────────
       2026-09-22            2026-10-23

sleep  ████████████████████████  28.7s  78 runs
go     ▇                          1.1s  208 runs
git    ▂                         318ms  130 runs
head   ▂                         306ms  78 runs
echo   ▁                         113ms  130 runs

| Command | Runs | Total | Avg | Max | Failed | Last |
| --- | --- | --- | --- | --- | --- | --- |
| sleep | 78 | 28.7s | 367ms | 574ms | 5 | just now |
| go version | 130 | 686ms | 5ms | 11ms | 4 | just now |
| go env GOOS | 78 | 379ms | 4ms | 11ms | 1 | just now |
```

The daily chart answers whether a month was steady or was one Tuesday. Each column is a day, every bar is measured from zero, and the only value labelled is the busiest day: a number on every column would hide the shape. A day nothing ran is an empty column rather than a missing one.

The tool ranking rolls up every form of a tool, so `go build`, `go test` and `go vet` count as go. It is what the command table cannot show: a tool that runs two hundred times for a second in total against one that runs twice for half a minute.

On a terminal all three are drawn in colour, and the table gets the borders `atkins --help` draws its tables with. Redirected to a file or a pipe the output is plain, so the same command serves a terminal and a document with no flag to say which.

## Reading it from a program

`--json` and `--yaml` emit the whole report, with the durations in milliseconds rather than words, and a row per tool per day as the series a chart of tool use over time is drawn from.

```bash
atkins --stats --json
```

```json
{
  "database": "/w/atkins/.atkins/atkins.db",
  "days": 31,
  "from": "2026-09-22",
  "to": "2026-10-23",
  "runs": 624,
  "total_ms": 30512,
  "commands": [
    {
      "command": "sleep",
      "tool": "sleep",
      "runs": 78,
      "total_ms": 28700,
      "average_ms": 367,
      "max_ms": 574,
      "failures": 5,
      "last_run_at": "2026-10-23T13:41:02.118Z"
    }
  ],
  "tools": [
    { "tool": "sleep", "commands": 1, "runs": 78, "total_ms": 28700, "failures": 5 }
  ],
  "usage": [
    { "date": "2026-09-25", "tool": "sleep", "runs": 3, "total_ms": 1104 },
    { "date": "2026-09-25", "tool": "go", "runs": 10, "total_ms": 52 }
  ]
}
```

A day a tool did not run has no row in `usage`, so a reader drawing a series fills the gaps with zero.

## Where the file is

Atkins walks up from the working directory looking for a `.atkins` folder and records into the `atkins.db` inside the first one it finds, so every run in a checkout counts towards the same history whichever subdirectory it started in. A directory that belongs to no project records into `$HOME/.atkins/atkins.db`, so a stray command in `/tmp` is counted too.

Point it somewhere else with `storage.database` in `.atkins/config.yml`, or with `ATKINS_DB`.

## What a run records

Two tables, because a job and a command are different questions.

The `run` table holds one row per node of the tree atkins walked: the pipeline, its jobs, their steps. Each row carries the time that node took with everything underneath it. A `default` job that runs `sleep 1` and `sleep 2` records three seconds, and the two steps record one and two.

The `command` table holds one row per command a step executed, with the command as the shell received it, its exit code, its directory and how long it took. A `$(...)` substitution is a command too, and records as one.

Both tables carry the `invocation_id` of the pipeline row, so the rows of one `atkins` process are one group. The pipeline row also carries the file, the module, the repository, the branch and the commit, so a history can be read per branch.

## The normalised command

A command is also stored in a normalised form, which is what the statistics group by. The form keeps the tool and its subcommands, keeps the flags with their values removed, sorts the flags, and drops everything else:

| Command                             | Normalised              |
|-------------------------------------|-------------------------|
| `go test ./...`                     | `go test`               |
| `go test -v ./...`                  | `go test -v`            |
| `go test -cover -v ./...`           | `go test -cover -v`     |
| `go test -v -cover ./...`           | `go test -cover -v`     |
| `go test -coverprofile=c.out ./...` | `go test -coverprofile` |
| `go test --cpuprofile cpu.out`      | `go test --cpuprofile`  |
| `CGO_ENABLED=0 go build -o bin/x .` | `go build -o`           |
| `sleep 1`                           | `sleep`                 |
| `head -20 README.md`                | `head`                  |

Sorting the flags is what makes `go test -v -cover` and `go test -cover -v` one row rather than two. Dropping the operands is what makes a flag's argument disappear without a table of which flags take one: in `--cpuprofile cpu.out` the flag stays and the file is an operand like any other. A flag that is only digits goes with them, so `head -20` and `head -50` count together.

A different flag set is a different row, on purpose: `go test` and `go test -race` are not the same thing to measure.

## Not recording

```bash
atkins --local build
```

`--local` runs without touching the database. `storage.record: false` in the configuration turns recording off for every run in the project, and `ATKINS_RECORD=false` for one shell.

A database that cannot be opened is not a reason to refuse to run. Atkins says so once and runs the pipeline unrecorded.

## Retention

A recorded run is kept for 31 days, measured from when it started, on every machine. It is not a setting: 31 days is long enough to hold a month of work and compare this week against the last, and short enough that a checkout nobody prunes does not grow without bound.

The run that records is the run that prunes. Each write drops whatever has fallen outside the window, in bounded batches, so the window holds with no daemon to hold it and no job to remember to run.

## Other readers

The file is an ordinary SQLite database and nothing else holds it open between runs, so anything may read it. [task-ui](https://github.com/titpetric/task-ui) reads it to chart tool usage over time.

## See Also

- [CLI Flags](./cli-flags) - `--stats`, `--local` and `--config`
- [Automation](./automation) - the JSON and YAML a run can emit instead
