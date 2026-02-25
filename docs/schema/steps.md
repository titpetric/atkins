---
title: Steps
subtitle: Step configuration reference
layout: page
---

# Steps Schema

Steps are individual commands or actions within a job.

## Basic Structure

```yaml
jobs:
  build:
    steps:
      - run: go build ./...
      - run: go test ./...
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name for the step |
| `run` | string | Shell command to execute |
| `cmd` | string | Alternative to `run` |
| `cmds` | list | Multiple commands in one step |
| `task` | string | Invoke another job/task |
| `dir` | string | Working directory |
| `if` | string | Condition for execution |
| `for` | string | Loop expression |
| `vars` | map | Step-scoped variables |
| `env` | object | Step-scoped environment |
| `detach` | bool | Run in background |
| `deferred` | bool | Run at end (cleanup) |
| `passthru` | bool | Show output with tree indentation |
| `tty` | bool | Allocate PTY (enables colors) |
| `interactive` | bool | Enable stdin |
| `summarize` | bool | Collapse output after completion |

## Command Formats

### Simple String

```yaml
steps:
  - echo hello
  - go build ./...
```

### Run Field

```yaml
steps:
  - run: echo hello
  - run: go build ./...
```

### Cmd Field

```yaml
cmds:
  - cmd: echo hello
  - cmd: go build ./...
```

### Multiple Commands

```yaml
steps:
  - cmds:
      - echo "Step 1"
      - echo "Step 2"
      - echo "Step 3"
```

### Multi-line Scripts

```yaml
steps:
  - run: |
      echo "Line 1"
      echo "Line 2"
      if [ -f config.yml ]; then
        echo "Config exists"
      fi
```

## Task Invocation

Call another job from a step:

```yaml
jobs:
  all:
    steps:
      - task: lint
      - task: test
      - task: build

  lint:
    steps:
      - run: golangci-lint run

  test:
    steps:
      - run: go test ./...

  build:
    steps:
      - run: go build ./...
```

### Cross-Pipeline References

Reference tasks from other pipelines using `:` prefix:

```yaml
steps:
  # Reference main pipeline's build task
  - task: :build

  # Reference go skill's test task
  - task: :go:test
```

## For Loops

Iterate over collections:

```yaml
vars:
  files:
    - main.go
    - util.go
    - helper.go

jobs:
  lint_all:
    steps:
      - for: file in files
        run: golint ${{ file }}
```

### Loop with Task

```yaml
vars:
  environments:
    - dev
    - staging
    - prod

jobs:
  deploy_all:
    steps:
      - for: env in environments
        task: deploy

  deploy:
    requires: [env]
    steps:
      - run: kubectl apply -f deploy/${{ env }}/
```

## Conditional Execution

```yaml
steps:
  - name: Deploy to production
    if: branch == "main"
    run: ./deploy.sh prod

  - name: Deploy to staging
    if: branch == "develop"
    run: ./deploy.sh staging
```

## Deferred Steps (Cleanup)

Deferred steps run at the end, even if earlier steps fail:

```yaml
steps:
  - defer:
      run: docker compose down

  - run: docker compose up -d
  - run: npm test
  # docker compose down runs here, regardless of test result
```

Or using the `deferred` flag:

```yaml
steps:
  - run: docker compose down
    deferred: true

  - run: docker compose up -d
  - run: npm test
```

## Working Directory

```yaml
steps:
  - dir: ./frontend
    run: npm install

  - dir: ./backend
    run: go build
```

## Environment Variables

```yaml
steps:
  - env:
      vars:
        GOOS: linux
        GOARCH: amd64
    run: go build -o app-linux
```

## Step Variables

```yaml
steps:
  - vars:
      output: ./bin/app
    run: go build -o ${{ output }}
```

## Background Execution

```yaml
steps:
  - run: ./server &
    detach: true

  - run: sleep 2  # Wait for server

  - run: curl localhost:8080/health
```

## Output Display

### Passthru Mode

Show output indented within the tree:

```yaml
steps:
  - run: go test -v ./...
    passthru: true
```

### TTY Mode

Enable color output for commands that detect terminals:

```yaml
steps:
  - run: npm test
    tty: true
```

### Interactive Mode

Allow keyboard input:

```yaml
steps:
  - run: ./setup-wizard.sh
    interactive: true
```

### Summarize Mode

Collapse output after completion:

```yaml
steps:
  - run: npm install
    summarize: true  # Shows summary instead of full output
```

## Named Steps

```yaml
steps:
  - name: Install dependencies
    run: go mod download

  - name: Run tests
    run: go test ./...

  - name: Build binary
    run: go build -o app
```

## Complete Example

```yaml
jobs:
  ci:
    steps:
      - name: Setup
        defer:
          run: docker compose down

      - name: Start services
        run: docker compose up -d

      - name: Wait for services
        run: sleep 5

      - name: Run tests
        run: go test -v ./...
        passthru: true
        tty: true

      - name: Build
        if: status == "success"
        vars:
          version: $(git describe --tags)
        env:
          vars:
            CGO_ENABLED: "0"
        run: go build -ldflags="-X main.Version=${{ version }}" -o app

      - name: Deploy
        if: branch == "main" && status == "success"
        run: ./deploy.sh
```
