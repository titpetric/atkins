---
title: Statistics
subtitle: The event database and what it answers
layout: page
---

Every run records what it did in a SQLite file, and `atkins --stats` reports the commands by the time they have taken altogether. The question it answers is where a repository's day goes: not which command is slowest, but which one costs the most once its frequency is counted.

```bash
atkins --stats
```

On a terminal the table is drawn with borders and colour, the way `atkins --help` draws its tables. Redirected to a file or a pipe it is plain markdown, so the same command serves a terminal and a document with no flag to say which:

```
/w/atkins/.atkins/atkins.db

| Command | Runs | Total | Avg | Max | Failed | Last |
| --- | --- | --- | --- | --- | --- | --- |
| go test -coverprofile | 128 | 1h14m | 34.8s | 1m12s | 6 | 12m ago |
| go build -o | 96 | 9m20s | 5.8s | 11.4s | 0 | 12m ago |
| splint | 64 | 2m08s | 2.0s | 3.1s | 0 | 3h ago |
| mig migrate --apply | 12 | 4.1s | 341ms | 890ms | 0 | 2d ago |
```

`--json` and `--yaml` emit the same report for a program, with the durations in milliseconds rather than words:

```bash
atkins --stats --json
```

```json
[
  {
    "command": "go test -coverprofile",
    "tool": "go",
    "runs": 128,
    "total_ms": 4452000,
    "average_ms": 34781,
    "max_ms": 72000,
    "failures": 6,
    "last_run_at": "2026-10-23T09:12:04.318Z"
  }
]
```

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
