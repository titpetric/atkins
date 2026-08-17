---
title: CLI Flags
subtitle: Command-line options reference
layout: page
---

Atkins provides command-line flags to control which pipeline file to use, which jobs to run, and how output is formatted.

## Usage

```bash
atkins [flags] [job-names...]
atkins server [flags]
atkins worker [flags]
```

Running jobs is the default, so `atkins build` and `atkins run build` are the same command. `server` runs the CI/CD server and `worker` runs an agent that claims its jobs; see [CI/CD Server](./ci-cd).

## Flag Reference

### Choosing what runs

| Flag                  | Short | Description                                                 |
|-----------------------|-------|-------------------------------------------------------------|
| `--file`              | `-f`  | Pipeline file to use, instead of the discovered one         |
| `--working-directory` | `-w`  | Change to this directory before running                     |
| `--jail`              |       | Load skills from the project only, ignoring `$HOME/.atkins` |
| `--list`              | `-l`  | List the jobs and dependencies instead of running them      |
| `--lint`              |       | Check the pipeline for errors and exit                      |

### Output

| Flag        | Short | Description                                       |
|-------------|-------|---------------------------------------------------|
| `--json`    | `-j`  | Emit the run, or the listing, as JSON             |
| `--yaml`    | `-y`  | Emit the run, or the listing, as YAML             |
| `--final`   |       | Print the tree once at the end, without redrawing |
| `--log`     |       | Write an execution log to this file               |
| `--debug`   |       | Print interpolation, evaluation and timing detail |
| `--version` | `-v`  | Print the version and build information           |

### CI/CD server

| Flag         | Description                                                     |
|--------------|-----------------------------------------------------------------|
| `--login`    | Log in to a server, e.g. `--login https://ci.example.com`       |
| `--register` | Create an account on a server and log in                        |
| `--logout`   | Detach this machine from the server it last logged in to        |
| `--dispatch` | Hand this run to an agent; refuses a dirty or unpushed checkout |
| `--local`    | Run here without recording the run on the server                |

### Project setup

| Flag       | Description                                                        |
|------------|--------------------------------------------------------------------|
| `--config` | Open the configuration menu over `.atkins/config.yml`              |
| `--vendor` | Report the skills this repository uses from `$HOME/.atkins/skills` |
| `--write`  | With `--vendor`, write them into `.atkins/skills`                  |

### Agent

| Flag      | Short | Description                               |
|-----------|-------|-------------------------------------------|
| `--agent` |       | Start the interactive agent REPL          |
| `--exec`  | `-x`  | Run one prompt through the agent and exit |

## File Discovery

By default, Atkins auto-discovers pipeline files in this order:

1. `.atkins.yml`
2. `.atkins.yaml`
3. `atkins.yml`
4. `atkins.yaml`

Override with `-f`:

```bash
# Use a specific file
atkins -f ci/build.yml

# Use a Taskfile
atkins -f Taskfile.yml
```

## Running Jobs

```bash
# Run default job
atkins

# Run specific job
atkins build

# Run multiple jobs in sequence
atkins lint test build

# Run namespaced job
atkins go:test
```

## Listing Jobs

```bash
# List all jobs (interactive display)
atkins -l

# List as YAML (for scripting/LLMs)
atkins -l -y

# List as JSON
atkins -l -j
```

Example output with `-l`:

```text
My Project

* default:     Run all checks (depends_on: lint, test)
* build:       Build the application
* test:        Run tests
* lint:        Run linters

Aliases

* b:           (invokes: build)
```

## Linting

Validate pipeline syntax without running:

```bash
atkins --lint
```

Checks for:
- Missing job dependencies
- Invalid task references
- Ambiguous step definitions

## Output Modes

### Interactive Tree (Default)

Shows live progress with colors and status indicators:

```bash
atkins
```

### Final Only

Renders tree only after completion (useful for CI logs):

```bash
atkins --final
```

### JSON/YAML Output

For automation and tooling integration:

```bash
# Execution output as JSON
atkins --json

# Execution output as YAML
atkins --yaml

# List jobs as JSON (for LLM tool integration)
atkins -l -j
```

`--json` and `--yaml` are mutually exclusive.

## Logging

Log command execution details to a file:

```bash
atkins --log execution.log
```

The log includes:
- Command start/end times
- Exit codes
- Output captured
- Timing information

## Working Directory

Change to a directory before running:

```bash
atkins -w ./subproject
```

Equivalent to:

```bash
cd ./subproject && atkins
```

## Debug Mode

Enable verbose debug output:

```bash
atkins --debug
```

Shows:
- Variable interpolation
- Command evaluation
- Timing details

## Jail Mode

Restrict skill loading to project scope only:

```bash
atkins --jail
```

Without `--jail`:
- Loads skills from `.atkins/skills/`
- Also loads from `$HOME/.atkins/skills/`

With `--jail`:
- Only loads from `.atkins/skills/`
- Ignores global skills

## Vendoring Skills

Report the skills this repository uses from `$HOME/.atkins/skills`, and what copying them into `.atkins/skills` next to `.git` would change:

```bash
atkins --vendor
```

```text
Found 7 local skills.
Found usage for 3 skills.
  + docker  +45 -0 (new)
  ✓ go      (up to date)
  ~ mdox    +4 -1 (changed)
Would install: docker, mdox.
Run atkins --vendor --write to write them.
```

Nothing is written until `--write`, which creates the missing skills and overwrites a vendored copy that has drifted:

```bash
atkins --vendor --write
```

`--debug` adds the reason each skill was selected and the diff behind the line counts. The copies are ordinary repository content, so a clone or a CI agent gets the same jobs without a personal skills directory. `--vendor` and `--jail` cannot be combined: jail mode excludes the directory vendoring reads from. See [Skills](./skills#vendoring) for the selection rules.

## Agent REPL

```bash
atkins --agent                    # interactive
atkins -x "run the tests"         # one prompt, then exit
```

The agent reads the pipelines and skills this project resolves to and drives them for you. It is the same job resolution the flags above use; nothing is available to it that `atkins -l` doesn't show.

## Configuration

```bash
atkins --config
```

Opens a menu over `.atkins/config.yml`, creating it from the built-in defaults if it doesn't exist. The document is the source of truth for how this machine talks to a CI/CD server and how `atkins server` and `atkins worker` run; `ATKINS_*` variables overlay individual fields on top of it.

## CI/CD Server Login

These flags attach the machine to an atkins CI/CD server. Each takes over the invocation and exits; no pipeline runs.

```bash
# Create an account and log in (first account on a server becomes admin)
atkins --register https://ci.example.com

# Log in on any other machine
atkins --login https://ci.example.com

# Detach this machine
atkins --logout
```

Credentials are stored per server in `~/.atkins/credentials.json` (mode `0600`).

Once logged in, `atkins` still runs the pipeline **here** and records the run on the server: the job page ends up with the transcript, the commit and the exit code an agent would have reported. To hand a run to an agent instead:

```bash
atkins --dispatch     # prints one job URL and exits
```

`--dispatch` refuses a dirty or unpushed work tree, since an agent builds the commit rather than your disk. To run here and record nothing:

```bash
atkins --local        # this run only
```

See [CI/CD Server](./ci-cd) for the configuration, the API and how agents claim jobs.

## Combining Flags

Flags can be combined:

```bash
# List jobs from specific file as YAML
atkins -f ci/pipeline.yml -l -y

# Run with debug and logging
atkins --debug --log debug.log

# Lint a specific file
atkins -f Taskfile.yml --lint
```

## Shebang Execution

On Unix systems, pipeline files can be directly executable:

```yaml
#!/usr/bin/env atkins
name: My Script

tasks:
  default:
    steps:
      - run: echo "Hello!"
```

```bash
chmod +x script.yml
./script.yml
```

## Stdin Input

Pipelines can be piped via stdin:

```bash
cat pipeline.yml | atkins

# Or with here-doc
atkins <<EOF
tasks:
  default:
    steps:
      - run: echo "From stdin"
EOF
```

## See Also

- [Job Targeting](./job-targeting) - Running specific jobs
- [Script Mode](./script-mode) - Executable pipelines
- [Automation](./automation) - JSON/YAML output details
- [CI/CD Server](./ci-cd) - `--login`, `--register` and job dispatch
