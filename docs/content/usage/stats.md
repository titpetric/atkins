---
title: Statistics
subtitle: The event database and what it answers
layout: page
---

Every run records what it did in a SQLite file, and `atkins --stats` reports the commands by the time they have taken altogether. The question it answers is where a repository's day goes: not which command is slowest, but which one costs the most once its frequency is counted.

```bash
atkins --stats
```

The report is a section per database, the project first, each with three parts: a column per day of the window, the tools ranked by the time they have taken, and the commands in a table. A project section names the repository its runs came from.

```
project · github.com/titpetric/atkins  /w/atkins/.atkins/atkins.db
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

| Command | Runs | Total | Avg | Max | Failed | Last |
| --- | --- | --- | --- | --- | --- | --- |
| sleep | 78 | 28.7s | 367ms | 574ms | 5 | just now |
| go version | 130 | 686ms | 5ms | 11ms | 4 | just now |
| go env GOOS | 78 | 379ms | 4ms | 11ms | 1 | just now |

workspace  /home/you/.atkins/atkins.db
31 days, 2026-09-22 to 2026-10-23, 4m12s across 3104 runs
...
```

The daily chart answers whether a month was steady or was one Tuesday. Each column is a day, every bar is measured from zero, and the only value labelled is the busiest day: a number on every column would hide the shape. A day nothing ran is an empty column rather than a missing one.

The tool ranking rolls up every form of a tool, so `go build`, `go test` and `go vet` count as go. It is what the command table cannot show: a tool that runs two hundred times for a second in total against one that runs twice for half a minute.

On a terminal all three are drawn in colour, and the table gets the borders `atkins --help` draws its tables with. Redirected to a file or a pipe the output is plain, so the same command serves a terminal and a document with no flag to say which.

## Reading it from a program

`--json` and `--yaml` emit the whole report, with the durations in milliseconds rather than words: a row per tool per day in `usage` and a row per command key per day in `command_usage`, the series a chart of tool or command use over time is drawn from.

```bash
atkins --stats --json
```

A scope per database, keyed by what it is. The `project` key is absent for a directory with no project of its own.

```json
{
  "project": {
    "name": "project",
    "repository": "github.com/titpetric/atkins",
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
    ],
    "command_usage": [
      { "date": "2026-09-25", "command": "sleep", "tool": "sleep", "runs": 3, "total_ms": 1104 },
      { "date": "2026-09-25", "command": "go version", "tool": "go", "runs": 6, "total_ms": 31 }
    ]
  },
  "workspace": {
    "name": "workspace",
    "database": "/home/you/.atkins/atkins.db",
    "days": 31,
    "from": "2026-09-22",
    "to": "2026-10-23",
    "runs": 3104,
    "total_ms": 252018,
    "commands": [],
    "tools": [],
    "usage": [],
    "command_usage": []
  }
}
```

The two scopes carry the same fields, so a reader draws either with one piece of code. `repository` is on the project alone: a workspace holds many and names none.

A day a tool or a command did not run has no row in `usage` or `command_usage`, so a reader drawing a series fills the gaps with zero.

## Two databases

A run records twice, because the two answer different questions.

The **project's** is the `atkins.db` of the nearest `.atkins` folder, found by walking up from the directory atkins runs in. Every run in a checkout counts towards the same history, whichever subdirectory it started in, and it says what standing in that checkout costs.

The **workspace's** is `$HOME/.atkins/atkins.db`, which every run records to. It holds every project on the machine together, and says where a week went across all of them. A command run in a directory that belongs to no project lands here alone, so a stray `sleep` in `/tmp` is counted too.

A directory inside `$HOME` whose nearest `.atkins` folder is the workspace's own has no project of its own. There is one database, and the run records to it once.

| Where atkins runs                    | Project                       | Workspace                 |
|--------------------------------------|-------------------------------|---------------------------|
| `/w/atkins`, holding `.atkins/`      | `/w/atkins/.atkins/atkins.db` | `$HOME/.atkins/atkins.db` |
| `/w/atkins/runner`, under it         | `/w/atkins/.atkins/atkins.db` | `$HOME/.atkins/atkins.db` |
| `/tmp/scratch`, under no project     | none                          | `$HOME/.atkins/atkins.db` |
| `$HOME/notes`, under `$HOME/.atkins` | none                          | `$HOME/.atkins/atkins.db` |

`storage.database` in `.atkins/config.yml`, or `ATKINS_DB`, names one database and makes it the only one. Naming a file says where the history goes, and a second copy elsewhere would make that less true.

## What a run records

Two tables, because a job and a command are different questions.

The `run` table holds one row per node of the tree atkins walked: the pipeline, its jobs, their steps. Each row carries the time that node took with everything underneath it. A `default` job that runs `sleep 1` and `sleep 2` records three seconds, and the two steps record one and two.

The `command` table holds one row per command a step executed, with the command as the shell received it, its exit code, its directory and how long it took. A `$(...)` substitution is a command too, and records as one.

Both tables carry the `invocation_id` of the pipeline row, so the rows of one `atkins` process are one group. The pipeline row also carries the file, the module, the repository, the branch and the commit, so a history can be read per branch.

## The command key

The report groups commands by their key: the words up to the first argument that starts with `-`, at most three of them, and none that is a path or a value. A word holding `/`, `.`, `=`, `$`, a quote or another shell character ends the key, and so does a number. The key leaves out an environment assignment ahead of the tool, and a pipeline or a list keys by its first command.

| Command                                 | Key                  |
|-----------------------------------------|----------------------|
| `go test ./...`                         | `go test`            |
| `go test -v -cover ./...`               | `go test`            |
| `docker build -q -t app .`              | `docker build`       |
| `docker compose up -d`                  | `docker compose up`  |
| `git clone --depth 1 <url>`             | `git clone`          |
| `git rev-parse HEAD \| cut -c1-8`       | `git rev-parse HEAD` |
| `kubectl get pods web-1 -o yaml`        | `kubectl get pods`   |
| `npm run build`                         | `npm run build`      |
| `cat README.md`                         | `cat`                |
| `CGO_ENABLED=0 go build -o bin/x .`     | `go build`           |

Shell builtins a step runs to set itself up, `set`, `cd`, `export`, `echo`, `test`, `[`, `true`, `false` and `:`, are neither recorded nor reported. `set -eu` at the top of a script is not a tool.

The database also stores each command in a normalised form, the tool and its subcommands with the flags sorted and their values removed, such as `go test -cover -v`. The report folds those forms into their keys as it reads them, so the rows a run recorded before the key existed count under it too.

## Not recording

```bash
atkins --local build
```

`--local` runs without touching the database. `storage.record: false` in the configuration turns recording off for every run in the project, and `ATKINS_RECORD=false` for one shell.

A database that cannot be opened is not a reason to refuse to run. Atkins says so once and runs the pipeline unrecorded.

## Retention

A recorded run is kept for 31 days, measured from when it started, on every machine. It is not a setting: 31 days is long enough to hold a month of work and compare this week against the last, and short enough that a checkout nobody prunes does not grow without bound.

The run that records is the run that prunes, in both databases. Each write drops whatever has fallen outside the window, in bounded batches, so the window holds with no daemon to hold it and no job to remember to run. The workspace database takes every project's rows, and 31 days is what keeps it a month of history rather than a year of it.

## Other readers

Both files are ordinary SQLite databases and nothing holds them open between runs, so anything may read them. [task-ui](https://github.com/titpetric/task-ui) reads them to chart tool usage over time, a project's beside the workspace's.

## See Also

- [CLI Flags](./cli-flags) - `--stats`, `--local` and `--config`
- [Automation](./automation) - the JSON and YAML a run can emit instead
