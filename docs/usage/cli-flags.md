---
title: CLI Flags
subtitle: Command-line options reference
layout: page
---

# CLI Flags

Atkins provides several command-line flags to control execution and output.

## Usage

```bash
atkins [flags] [job-name]
```

## Flag Reference

| Flag | Short | Description |
|------|-------|-------------|
| `--file` | `-f` | Path to pipeline file |
| `--job` | | Specific job to run |
| `--list` | `-l` | List available jobs |
| `--lint` | | Validate pipeline syntax |
| `--json` | `-j` | Output in JSON format |
| `--yaml` | `-y` | Output in YAML format |
| `--final` | | Show only final tree (no live updates) |
| `--log` | | Log execution to file |
| `--debug` | | Enable debug output |
| `--working-directory` | `-w` | Change directory before running |
| `--jail` | | Restrict to project scope only |

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

# Run job with --job flag
atkins --job build

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

```
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

Note: `--json` and `--yaml` are mutually exclusive.

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
