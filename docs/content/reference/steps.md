---
title: Steps
subtitle: Step schema reference
layout: page
---

Steps are the individual commands or actions within a job.

## Properties

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | - | Step name for display |
| `run` | string | - | Command to execute |
| `cmd` | string | - | Alias for `run` |
| `cmds` | list | - | Multiple commands |
| `task` | string | - | Task/job to invoke |
| `if` | string | - | Conditional execution |
| `for` | string | - | Loop iteration |
| `requires` | list | - | Required loop variables |
| `dir` | string | - | Working directory |
| `env` | object | - | Step environment |
| `timeout` | string | - | Execution timeout |
| `deferred` | bool | `false` | Run on cleanup |
| `detach` | bool | `false` | Run in background |
| `tty` | bool | `false` | Allocate PTY |
| `interactive` | bool | `false` | Live streaming |
| `verbose` | bool | `false` | Show output |
| `quiet` | bool | `false` | Suppress output |

## Basic Steps

@tabs
@file "Pipeline" steps/basic.yml

![](./steps/basic.png)

## Named Steps

@tabs
@file "Pipeline" steps/named.yml

![](./steps/named.png)

## Task Invocation

Call other jobs using `task:`:

@tabs
@file "Pipeline" steps/task-invoke.yml

![](./steps/task-invoke.png)

## Deferred Steps

Steps with `deferred: true` run after the job completes (like `defer` in Go):

@tabs
@file "Pipeline" steps/deferred.yml

![](./steps/deferred.png)

## For Loops

Iterate over lists with `for:`:

@tabs
@file "Pipeline" steps/for-loop.yml

![](./steps/for-loop.png)

## Conditional Steps

Execute steps conditionally using `if`:

@tabs
@file "Pipeline" steps/conditional.yml

![](./steps/conditional.png)

## Step Environment

Override environment for a single step:

@tabs
@file "Pipeline" steps/with-env.yml

![](./steps/with-env.png)

## See Also

- [Jobs](./jobs) - Job configuration
- [Variables](./variables) - Variable interpolation
