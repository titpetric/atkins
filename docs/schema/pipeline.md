---
title: Pipeline
subtitle: Root pipeline configuration
layout: page
---

# Pipeline Schema

The pipeline is the root configuration in an `atkins.yml` file.

## Basic Structure

```yaml
name: My Pipeline

vars:
  key: value

jobs:       # or 'tasks:'
  default:
    steps:
      - run: echo hello
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Pipeline name displayed in output |
| `vars` | map | Variables available to all jobs |
| `env` | object | Environment variables for all jobs |
| `include` | list | Files to include for additional variables |
| `jobs` | map | Job definitions (GHA-style) |
| `tasks` | map | Task definitions (Taskfile-style) |
| `when` | object | Conditions for skill activation |

## Jobs vs Tasks

`jobs` and `tasks` are functionally identical - use whichever style you prefer:

```yaml
# GitHub Actions style
jobs:
  build:
    steps:
      - run: go build

# Taskfile style
tasks:
  build:
    cmds:
      - go build
```

If both are defined, `jobs` takes precedence.

## Pipeline Variables

Variables declared at the pipeline level are available to all jobs:

```yaml
vars:
  version: 1.0.0
  build_flags: -ldflags="-s -w"

jobs:
  build:
    steps:
      - run: go build ${{ build_flags }} -o app-${{ version }}
```

## Environment Variables

Set environment variables for all jobs:

```yaml
env:
  vars:
    GOOS: linux
    GOARCH: amd64

jobs:
  build:
    steps:
      - run: go build  # Uses GOOS and GOARCH
```

## Include Files

Load variables from external files:

```yaml
include:
  - config/vars.yml
  - secrets.yml

jobs:
  deploy:
    steps:
      - run: deploy --token ${{ api_token }}
```

## Skills and When Conditions

Skills are pipelines that activate based on conditions:

```yaml
# .atkins/skills/go.yml
name: Go Skill
when:
  files:
    - go.mod

jobs:
  build:
    steps:
      - run: go build ./...
```

The `when.files` condition activates this skill when `go.mod` exists.

## Multi-Document Pipelines

A single file can contain multiple pipelines using YAML document separators:

```yaml
name: Main Pipeline

jobs:
  default:
    steps:
      - run: echo "Main"
---
name: Secondary Pipeline

jobs:
  other:
    steps:
      - run: echo "Other"
```

## Example: Complete Pipeline

```yaml
name: My Project CI

vars:
  go_version: "1.22"
  output_dir: ./bin

env:
  vars:
    CGO_ENABLED: "0"

include:
  - build-config.yml

jobs:
  default:
    desc: Run all checks
    depends_on: [lint, test, build]

  lint:
    desc: Run linters
    detach: true
    steps:
      - run: golangci-lint run

  test:
    desc: Run tests
    detach: true
    steps:
      - run: go test ./...

  build:
    desc: Build binary
    steps:
      - run: go build -o ${{ output_dir }}/app ./...
```
