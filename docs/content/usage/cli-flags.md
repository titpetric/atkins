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

| Flag                  | Short | Description                            |
|-----------------------|-------|----------------------------------------|
| `--file`              | `-f`  | Path to pipeline file                  |
| `--list`              | `-l`  | List available jobs                    |
| `--lint`              |       | Validate pipeline syntax               |
| `--json`              | `-j`  | Output in JSON format                  |
| `--yaml`              | `-y`  | Output in YAML format                  |
| `--final`             |       | Show only final tree (no live updates) |
| `--log`               |       | Log execution to file                  |
| `--debug`             |       | Enable debug output                    |
| `--version`           | `-v`  | Print version and build information    |
| `--working-directory` | `-w`  | Change directory before running        |
| `--jail`              |       | Restrict to project scope only         |
| `--config`            |       | Open the configuration menu            |
| `--login`             |       | Log in to a CI/CD server               |
| `--register`          |       | Register an account on a CI/CD server  |
| `--logout`            |       | Log out of the CI/CD server            |
| `--local`             |       | Run here instead of dispatching        |

### Listing a pipeline that does not lint

Both `--lint` and `--list` check the pipeline before they do anything with it, and they treat the answer differently.

`--lint` is asked whether the pipeline is sound, so the first job that does not resolve ends it. `--list` is asked what is in the pipeline, and one broken job is no reason to withhold the rest: it lists everything, then prints the lint errors as a warning **on stderr**, and exits non-zero.

The two halves are both deliberate. The warning is on stderr so that `atkins --list --json > jobs.json` still writes a document a reader can parse — the lint errors do not land in the middle of it. The exit status is still a failure, because a listing of a pipeline that does not resolve is a best effort at a broken thing, and a caller that only reads the status should not be told otherwise.

```bash
$ atkins --list --json > jobs.json
⚠ WARNING: Pipeline 'my project' has errors:
  test:docs: step references task 'mdox:fmt', but job "mdox:fmt" not found
Error: 1 job did not lint; it is listed above, but cannot run

$ echo $?
1
$ jq '[.[].cmds[]] | length' jobs.json    # the rest of the pipeline is still there
40
```

A script that wants the listing regardless should read the file rather than the exit status, and check that it is non-empty — an `atkins` that could not list at all writes nothing.

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

Once logged in, `atkins` **hands runs to the server** instead of running them: it prints one job URL and exits, and an agent runs the pipeline against a fresh checkout. To run here instead:

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
