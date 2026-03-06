---
title: Steps
subtitle: Step configuration and execution
layout: page
---

Steps are the individual commands or actions within a job. They execute sequentially by default.

## Step Fields

| Field | Description |
|-------|-------------|
| `run:` | Shell command to execute |
| `cmd:` | Alias for `run:` |
| `cmds:` | List of commands (run sequentially) |
| `task:` | Invoke another job/task by name |
| `name:` | Display name for the step |
| `if:` | Conditional execution |
| `for:` | Loop iteration (`for: item in collection`) |
| `dir:` | Working directory |
| `detach: true` | Run step in background |
| `deferred: true` | Run after other steps complete |
| `defer:` | Shorthand for a deferred step |
| `verbose: true` | Show more output |
| `passthru: true` | Output with tree indentation |
| `tty: true` | Allocate a PTY for color output |
| `interactive: true` | Live streaming with stdin |
| `vars:` | Step-level variables |
| `env:` | Step-level environment variables |

## Example

```yaml
jobs:
  setup:
    steps:
      - name: Install dependencies
        run: npm install
      - name: Run migrations
        run: npm run migrate
      - name: Start server
        run: npm start
        detach: true
```

## Task Invocation

Reference other jobs from a step:

```yaml
jobs:
  default:
    steps:
      - task: lint
      - task: build
      - task: test

  lint:
    steps:
      - run: golangci-lint run

  build:
    steps:
      - run: go build .
```

## Deferred Steps

Deferred steps run after all other steps complete, regardless of success or failure. Useful for cleanup:

```yaml
steps:
  - defer:
      run: docker compose down
  - run: docker compose up -d
  - run: go test ./...
```

## For Loops

Invoke tasks repeatedly with different loop variables:

```yaml
vars:
  environments:
    - dev
    - staging
    - prod

tasks:
  deploy_all:
    steps:
      - for: env in environments
        task: deploy_service

  deploy_service:
    requires: [env]
    steps:
      - run: kubectl apply -f config/${{ env }}/deployment.yml
```

See [Loops](./loops) for advanced loop patterns.

## See Also

- [Pipelines](./pipelines) - Pipeline-level configuration
- [Jobs](./jobs) - Job configuration and dependencies
- [Loops](./loops) - For loop details
- [Conditionals](./conditionals) - Conditional execution
