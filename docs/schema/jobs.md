---
title: Jobs & Tasks
subtitle: Defining jobs and tasks
layout: page
---

# Jobs & Tasks Schema

Jobs (or tasks) are the primary execution units in Atkins.

## Basic Structure

```yaml
jobs:
  build:
    desc: Build the application
    steps:
      - run: go build ./...
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `desc` | string | Description shown in listings |
| `steps` | list | Steps to execute (GHA-style) |
| `cmds` | list | Commands to execute (Taskfile-style) |
| `cmd` | string | Single command shorthand |
| `run` | string | Single run command shorthand |
| `dir` | string | Working directory for the job |
| `if` | string | Condition for execution |
| `depends_on` | string/list | Jobs that must complete first |
| `aliases` | list | Alternative names for this job |
| `requires` | list | Required variables (for loop invocation) |
| `detach` | bool | Run in background |
| `timeout` | string | Maximum execution time |
| `passthru` | bool | Show output with tree indentation |
| `tty` | bool | Allocate PTY (enables colors) |
| `interactive` | bool | Enable stdin for keyboard input |
| `summarize` | bool | Collapse output after completion |
| `vars` | map | Job-scoped variables |
| `env` | object | Job-scoped environment |

## Shorthand Syntax

For simple single-command jobs:

```yaml
tasks:
  up: docker compose up -d
  down: docker compose down
  logs: docker compose logs -f
```

This is equivalent to:

```yaml
tasks:
  up:
    cmd: docker compose up -d
    passthru: true
  down:
    cmd: docker compose down
    passthru: true
```

## Steps vs Cmds

Both are functionally identical:

```yaml
# GHA-style
jobs:
  build:
    steps:
      - run: go build

# Taskfile-style
tasks:
  build:
    cmds:
      - go build
```

## Dependencies

Jobs can depend on other jobs:

```yaml
jobs:
  lint:
    steps:
      - run: golangci-lint run

  test:
    steps:
      - run: go test ./...

  build:
    depends_on: [lint, test]  # Waits for both
    steps:
      - run: go build ./...
```

Single dependency:

```yaml
jobs:
  deploy:
    depends_on: build
    steps:
      - run: ./deploy.sh
```

## Parallel Execution

Use `detach: true` to run jobs in parallel:

```yaml
jobs:
  lint:
    detach: true
    steps:
      - run: golangci-lint run

  test:
    detach: true
    steps:
      - run: go test ./...

  build:
    depends_on: [lint, test]
    steps:
      - run: go build
```

## Conditional Execution

```yaml
jobs:
  deploy:
    if: branch == "main" && status == "success"
    steps:
      - run: ./deploy.sh
```

Conditions use [expr-lang](https://expr-lang.org/) syntax.

## Aliases

Create shortcuts for jobs:

```yaml
jobs:
  docker:build:
    aliases: [build, b]
    steps:
      - run: docker build -t app .
```

Now `atkins build` or `atkins b` invokes `docker:build`.

## The Default Job

When running `atkins` without arguments, it looks for:

1. A job named `default`
2. A job with `default` in its `aliases`

```yaml
jobs:
  all:
    aliases: [default]
    depends_on: [lint, test, build]
```

## Nested Jobs

Jobs with `:` in their name are nested and not executed directly:

```yaml
jobs:
  build:
    steps:
      - task: build:linux
      - task: build:darwin

  build:linux:
    steps:
      - run: GOOS=linux go build -o bin/app-linux

  build:darwin:
    steps:
      - run: GOOS=darwin go build -o bin/app-darwin
```

`build:linux` and `build:darwin` only run when invoked by `build`.

## Job Variables

Define job-scoped variables:

```yaml
jobs:
  build:
    vars:
      output: ./bin/app
    steps:
      - run: go build -o ${{ output }}
```

## Working Directory

```yaml
jobs:
  frontend:
    dir: ./web
    steps:
      - run: npm install
      - run: npm run build
```

## Timeouts

```yaml
jobs:
  long_test:
    timeout: 30m
    steps:
      - run: go test -race ./...
```

## Output Display

```yaml
jobs:
  test:
    passthru: true    # Show output indented in tree
    tty: true         # Enable color output
    steps:
      - run: go test -v ./...
```

## Required Variables (For Loops)

When a job is called from a loop, declare required variables:

```yaml
vars:
  services:
    - api
    - web
    - worker

jobs:
  deploy_all:
    steps:
      - for: service in services
        task: deploy_service

  deploy_service:
    requires: [service]
    steps:
      - run: kubectl apply -f deploy/${{ service }}.yml
```
