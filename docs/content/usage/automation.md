---
title: Automation (JSON/YAML)
subtitle: Machine-readable output for tooling integration
layout: page
---

Atkins provides machine-readable output formats for integration with scripts, tools, and LLMs. The `--json` and `--yaml` flags suppress interactive output and produce structured data that can be parsed programmatically.

## Output Formats

### List Jobs as YAML

```bash
atkins -l -y
```

Output:

```yaml
- desc: My Project
  cmds:
    - id: default
      desc: Run everything
      cmd: atkins default
    - id: build
      desc: Build the app
      cmd: atkins build
- desc: Aliases
  cmds:
    - id: b
      desc: invokes build
      cmd: atkins b
```

### List Jobs as JSON

```bash
atkins -l -j
```

Output:

```json
[
  {
    "desc": "My Project",
    "cmds": [
      {
        "id": "default",
        "desc": "Run everything",
        "cmd": "atkins default"
      },
      {
        "id": "build",
        "desc": "Build the app",
        "cmd": "atkins build"
      }
    ]
  }
]
```

## List Output Schema

```yaml
- desc: string        # Pipeline/section name
  cmds:
    - id: string      # Job identifier (e.g., "build", "go:test")
      desc: string    # Job description (optional)
      cmd: string     # Full command to run this job
```

Sections in order:
1. Main pipeline jobs
2. Aliases
3. Skill pipeline jobs (one section per skill)

## Execution Output

### Run with YAML Output

```bash
atkins --yaml
```

Suppresses the interactive tree and prints the execution state as YAML once the run finishes.

### Run with JSON Output

```bash
atkins --json
```

Same behavior, but prints JSON. The two share one field set and the same snake_case keys; only the encoding differs.

### Example Execution Output

```bash
$ atkins --json build
```

```json
{
  "name": "My Project",
  "status": "passed",
  "result": "pass",
  "created_at": "2026-01-01T12:00:00Z",
  "updated_at": "2026-01-01T12:00:03Z",
  "children": [
    {
      "name": "build - Build the application",
      "status": "passed",
      "result": "pass",
      "created_at": "2026-01-01T12:00:00Z",
      "updated_at": "2026-01-01T12:00:03Z",
      "duration": 3.12,
      "children": [
        {
          "name": "go build ./...",
          "id": "jobs.build.steps.0",
          "status": "passed",
          "result": "pass",
          "created_at": "2026-01-01T12:00:00Z",
          "updated_at": "2026-01-01T12:00:03Z",
          "duration": 3.1
        }
      ]
    }
  ]
}
```

| Field        | Present when                                           |
|--------------|--------------------------------------------------------|
| `name`       | always                                                 |
| `id`         | a step, e.g. `jobs.build.steps.0`; empty on job nodes  |
| `status`     | always: `passed`, `failed`, `skipped` or `conditional` |
| `result`     | a leaf step ran: `pass`, `fail` or `skipped`           |
| `if`         | the node carried an `if:` condition                    |
| `created_at` | always                                                 |
| `updated_at` | the node's status changed after creation               |
| `start`      | nonzero: seconds since the run started                 |
| `duration`   | nonzero: seconds the node took                         |
| `steps`      | nonzero: steps executed under a job or workflow node   |
| `children`   | the node has any                                       |

## Use Cases

### LLM Tool Integration

Provide job listings to LLMs for intelligent task selection:

```bash
# Get available commands for LLM context
atkins -l -y > /tmp/commands.yml
```

The YAML format works well for LLMs: clear structure, includes executable commands, and human-readable descriptions.

### CI/CD Pipeline Discovery

```bash
# Parse available jobs in CI
JOBS=$(atkins -l -j | jq -r '.[0].cmds[].id')
for job in $JOBS; do
  echo "Available: $job"
done
```

### Script Integration

```bash
#!/bin/bash
# Run a job and capture structured output

OUTPUT=$(atkins build --json)
STATUS=$(echo "$OUTPUT" | jq -r '.status')

if [ "$STATUS" = "passed" ]; then
  echo "Build succeeded"
else
  echo "Build failed"
  exit 1
fi
```

### Monitoring and Logging

```bash
# Log execution with structured data
atkins --json --log execution.log > result.json

# Process results
jq '.duration' result.json
```

## Combining with Other Flags

```bash
# List specific file's jobs as JSON
atkins -f ci/pipeline.yml -l -j

# Run with final-only display and JSON output
atkins --final --json

# Debug mode with JSON output
atkins --debug --json
```

## Flag Constraints

- `--json` and `--yaml` are mutually exclusive
- Both suppress the interactive tree display
- Both work with `-l` (list) and execution modes

```bash
# Error: flags cannot be combined
atkins -l -j -y
```

## Practical Examples

### Build Dashboard Integration

```bash
# Fetch job list for dashboard
curl -X POST https://dashboard.example.com/api/projects \
  -H "Content-Type: application/json" \
  -d "$(atkins -l -j)"
```

### Slack Notification

```bash
# Run and notify on failure
RESULT=$(atkins build --json)
STATUS=$(echo "$RESULT" | jq -r '.status')
DURATION=$(echo "$RESULT" | jq -r '.duration')

if [ "$STATUS" = "failed" ]; then
  curl -X POST https://slack.com/webhook \
    -d "{\"text\": \"Build failed after ${DURATION}s\"}"
fi
```

### GitHub Actions Integration

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Atkins
        run: go install github.com/titpetric/atkins@latest

      - name: Run Build
        run: |
          atkins build --json > result.json
          echo "status=$(jq -r '.status' result.json)" >> $GITHUB_OUTPUT
```

### Parallel Job Discovery

```bash
# Find all detachable jobs and run them
atkins -l -j | jq -r '.[].cmds[] | select(.id | contains(":")) | .cmd' | \
  parallel --jobs 4 {}
```

## Silent Mode Behavior

When using `--json` or `--yaml`:

1. Interactive tree is disabled
2. No progress output during execution
3. Only final state is printed to stdout
4. Errors still go to stderr
5. atkins exits 0 when every step passed, and otherwise with the exit code of the step that failed

This makes output parsing reliable without filtering ANSI codes or progress updates.

## See Also

- [CLI Flags](./cli-flags) - Command-line options
- [Script Mode](./script-mode) - Executable pipelines
- [CI/CD Server](./ci-cd) - Recording runs as jobs over an HTTP API
