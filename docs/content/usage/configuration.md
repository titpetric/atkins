---
title: Configuration
subtitle: Pipeline configuration format and syntax
layout: page
---

Atkins pipelines are defined in YAML files, typically named `atkins.yml` or `.atkins.yml`. A pipeline contains jobs (or tasks), each with a sequence of steps to execute. Atkins supports two syntax styles (Taskfile-compatible and GitHub Actions-inspired) which can be mixed in the same file.

This page covers the configuration format, variable interpolation, and key configuration concepts.

## Basic Structure

A minimal pipeline file:

```yaml
name: My Project

jobs:
  default:
    desc: Run everything
    steps:
      - run: echo "Hello, world!"
```

## Syntax Flavors

Atkins supports two syntax styles. Both can be used interchangeably within the same file.

### Taskfile-Style (`tasks` / `cmds`)

```yaml
name: My Project

tasks:
  default:
    desc: Run checks
    cmds:
      - go test ./...
      - go build ./...
```

Simple string shorthand for quick commands:

```yaml
tasks:
  up: docker compose up -d
  down: docker compose down
  ps: docker compose ps
  logs: docker compose logs -f
```

Steps can use `cmd:` for a single command or `cmds:` for multiple:

```yaml
tasks:
  build:
    cmd: go build ./...

  deploy:
    cmds:
      - go build -o bin/app .
      - scp bin/app server:/opt/app/
```

### GitHub Actions-Style (`jobs` / `steps`)

```yaml
name: My Project

jobs:
  default:
    desc: Build and test
    depends_on: [lint, test]
    steps:
      - name: Build
        run: go build ./...

  lint:
    steps:
      - run: golangci-lint run

  test:
    steps:
      - run: go test ./...
```

## Variable Interpolation

### `${{ expr }}` - Atkins Variables

Atkins uses `${{ expr }}` for variable interpolation. This syntax was chosen to avoid conflicts with bash `${var}}`. Both can coexist in the same command without escaping:

```yaml
vars:
  binary: myapp

jobs:
  build:
    steps:
      - run: go build -ldflags="-X 'main.Version=${GIT_TAG}'" -o bin/${{ binary }} .
```

Here `${GIT_TAG}` is resolved by the shell at runtime, while `${{ binary }}` is resolved by Atkins before execution.

### `$(command)` - Shell Execution

Shell command output can fill variable values:

```yaml
vars:
  commit: $(git rev-parse HEAD)
  branch: $(git rev-parse --abbrev-ref HEAD)
  tag: $(git describe --tags --always)

jobs:
  default:
    steps:
      - run: echo "Building ${{ branch }} at ${{ commit }}"
```

Shell execution works in `vars:`, `env:`, and inline values.

## Variables (`vars:`)

The `vars:` block defines pipeline-level variables. Values can be strings, lists, or shell-evaluated expressions.

```yaml
vars:
  app_name: myservice
  version: 1.2.3
  platforms:
    - linux
    - darwin
  commit: $(git rev-parse --short HEAD)
```

Jobs can define their own variables that merge with pipeline-level ones:

```yaml
vars:
  app_name: myservice

jobs:
  build:
    vars:
      output_dir: bin
    steps:
      - run: go build -o ${{ output_dir }}/${{ app_name }} .
```

Nested variable access uses dot notation:

```yaml
vars:
  build:
    goarch: [arm64, amd64]

jobs:
  compile:
    steps:
      - for: goarch in ${{ build.goarch }}
        run: GOARCH=${{ goarch }} go build -o bin/app-${{ goarch }} .
```

## Environment (`env:`)

The `env:` block sets environment variables. It can appear at the pipeline, job, or step level.

```yaml
env:
  vars:
    GIT_COMMIT: $(git rev-parse HEAD)
    GIT_BRANCH: $(git rev-parse --abbrev-ref HEAD)

jobs:
  build:
    steps:
      - run: echo "Commit is $GIT_COMMIT on $GIT_BRANCH"
```

Step-level environment overrides:

```yaml
jobs:
  install:
    steps:
      - env:
          vars:
            CGO_ENABLED: 0
            GOOS: linux
            GOARCH: amd64
        run: go build -o bin/app .
```

### Environment Inheritance

Atkins passes the existing shell environment through to all commands. There's no need to explicitly declare which variables to pass. Everything is inherited automatically. This differs from tools like Taskfile that require explicit environment declarations.

## Include (`include:`)

Compose pipelines from multiple files using `include:`:

```yaml
name: My Project

jobs:
  include: ci/*.yml
```

Each included file contributes its jobs to the pipeline, allowing large pipelines to be split into manageable pieces.

## Conditional Activation (`when:`)

The `when:` block controls when a skill activates based on project context. This is primarily used in [skill files](./skills).

```yaml
name: Go build and test

when:
  files:
    - go.mod

jobs:
  test:
    steps:
      - run: go test ./...
```

The skill activates only when `go.mod` exists. Multiple files use OR logic. Any match activates the skill. See [Skills](./skills) for details.

## Complete Example

```yaml
#!/usr/bin/env atkins
name: My App

env:
  vars:
    GIT_COMMIT: $(git rev-parse HEAD)
    GIT_TAG: $(git describe --tags --always)

vars:
  binary: myapp
  platforms:
    - amd64
    - arm64

jobs:
  default:
    desc: Run everything
    depends_on: fmt
    steps:
      - task: test
      - task: build

  fmt:
    desc: Format code
    steps:
      - run: gofmt -w .
      - run: go mod tidy

  test:
    desc: Run tests
    steps:
      - run: go test ./...

  build:
    desc: Cross-compile
    steps:
      - for: arch in platforms
        env:
          vars:
            CGO_ENABLED: 0
            GOARCH: ${{ arch }}
        run: go build -ldflags="-X 'main.Commit=${GIT_COMMIT}'" -o bin/${{ binary }}-${{ arch }} .
```

## See Also

- [Pipelines](./pipelines) - Pipeline-level configuration
- [Jobs](./jobs) - Job configuration and dependencies
- [Steps](./steps) - Step configuration and loops
- [CLI Flags](./cli-flags) - Command-line options
- [Job Targeting](./job-targeting) - Running specific jobs
