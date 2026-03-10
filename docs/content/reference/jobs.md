---
title: Jobs
subtitle: Job schema reference
layout: page
---

Jobs define units of work in a pipeline. Each job contains steps to execute.

Jobs can be defined using either `jobs:` (GitHub Actions style) or `tasks:` (Taskfile style). Both are interchangeable.

## Properties

| Field         | Type        | Default | Description                             |
|---------------|-------------|---------|-----------------------------------------|
| `desc`        | string      | -       | Short description for `--list`          |
| `steps`       | list        | `[]`    | Steps to execute                        |
| `cmds`        | list        | `[]`    | Alias for `steps`                       |
| `run`         | string      | -       | Single command (creates synthetic step) |
| `cmd`         | string      | -       | Alias for `run`                         |
| `depends_on`  | string/list | `[]`    | Jobs to run before this job             |
| `vars`        | map         | `{}`    | Job-level variables                     |
| `env`         | object      | `{}`    | Job-level environment                   |
| `include`     | string/list | -       | Include external files                  |
| `if`          | string      | -       | Conditional execution expression        |
| `dir`         | string      | -       | Working directory override              |
| `aliases`     | list        | `[]`    | Alternative names for invoking this job |
| `requires`    | list        | `[]`    | Variables required when invoked in loop |
| `timeout`     | string      | -       | Execution timeout (e.g., `10m`, `300s`) |
| `detach`      | bool        | `false` | Run in background                       |
| `show`        | bool        | auto    | Show in `--list` (root jobs shown)      |
| `summarize`   | bool        | `false` | Summarize output                        |
| `quiet`       | bool        | `false` | Suppress output                         |
| `passthru`    | bool        | `false` | Print output with tree indentation      |
| `tty`         | bool        | `false` | Allocate PTY for all steps              |
| `interactive` | bool        | `false` | Stream output live, connect stdin       |

## Basic Job

@tabs
@file "Pipeline" jobs/basic.yml

![](./jobs/basic.png)

## Job Dependencies

Jobs can depend on other jobs using `depends_on`:

@tabs
@file "Pipeline" jobs/dependencies.yml

![](./jobs/dependencies.png)

## Detached Jobs

Run jobs in the background with `detach: true`:

@tabs
@file "Pipeline" jobs/detached.yml

![](./jobs/detached.png)

## Conditional Jobs

Execute jobs conditionally using `if`:

@tabs
@file "Pipeline" jobs/conditional.yml

![](./jobs/conditional.png)

## String Shorthand

Jobs can be written as bare strings, useful for simple commands and skills:

```yaml
jobs:
  build: go build ./...
  test: go test ./...
  lint: golangci-lint run
```

This is equivalent to:

```yaml
jobs:
  build:
    desc: go build ./...
    steps:
      - go build ./...
```

@tabs
@file "Pipeline" jobs/shorthand.yml

![](./jobs/shorthand.png)

## Job Variables

Jobs can define their own variables that merge with pipeline-level ones:

@tabs
@file "Pipeline" jobs/with-vars.yml

![](./jobs/with-vars.png)

## See Also

- [Steps](./steps) - Step configuration
- [Variables](./variables) - Variable interpolation
