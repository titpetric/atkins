---
title: Migration from Task
subtitle: Moving from Taskfile to Atkins
layout: page
---

Atkins supports a Taskfile-compatible structure, making migration straightforward for most pipelines. The main differences are in interpolation syntax and shell substitution. Simple Taskfiles without interpolation often work without any changes.

This guide covers the syntax mappings and common patterns you'll encounter when migrating.

## Structure Comparison

The overall structure is nearly identical. Atkins doesn't require a `version` field.

**Taskfile:**
```yaml
version: '3'

vars:
  key: val

tasks:
  default:
    cmds:
      - cmd: echo hello
      - task: other
```

**Atkins:**
```yaml
vars:
  key: val

tasks:
  default:
    cmds:
      - cmd: echo hello
      - task: other
```

## Shell Substitution

Taskfile uses the `sh:` field for dynamic values. Atkins uses bash-style `$(...)` which can also be used inline within commands.

**Taskfile:**
```yaml
vars:
  uname:
    sh: uname -n
```

**Atkins:**
```yaml
vars:
  uname: $(uname -n)
```

## Variable Interpolation

Taskfile uses Go templates which require quoting in YAML. Atkins uses `${{ }}` syntax which is YAML-safe.

**Taskfile:**
```yaml
vars:
  foo: bar
  bar: "{{.foo}}"  # Must be quoted

tasks:
  default:
    cmds:
      - echo "{{.foo}}"
```

**Atkins:**
```yaml
vars:
  foo: bar
  bar: ${{ foo }}  # No quotes needed

tasks:
  default:
    cmds:
      - echo "${{ foo }}"
```

Benefits of the Atkins syntax:
- No quoting required in YAML
- No `.` prefix for variable names
- Doesn't conflict with bash `${var}` syntax

## Environment Variables

Taskfile requires explicit environment declarations and doesn't inherit from the shell by default. Atkins inherits the full environment automatically.

You can still declare environment variables at any level:

```yaml
env:
  vars:
    MY_VAR: value

tasks:
  build:
    env:
      vars:
        GOOS: linux
    steps:
      - run: go build
```

## Simple Tasks

Both support shorthand syntax for single-command tasks:

```yaml
tasks:
  up: docker compose up -d
  down: docker compose down
```

## Listing Tasks

**Taskfile:**
```bash
task --list-all  # -l requires descriptions
```

**Atkins:**
```bash
atkins -l  # Shows all tasks; uses command as description if none provided
```

Atkins always shows tasks, displaying the command when no description is set.

## What Works Directly

Simple Taskfiles without interpolation work identically:

```yaml
tasks:
  up: docker compose up -d --remove-orphans
  down: docker compose down --remove-orphans
```

## What Needs Changes

| Taskfile | Atkins |
|----------|--------|
| `sh: command` | `$(command)` |
| `{{.var}}` | `${{ var }}` |
| Quoted interpolation values | Often can remove quotes |

## Binary Size

Atkins is smaller than Task. The most notable dependency is `expr-lang` for evaluating `if` conditions. Task has heavier dependencies like syntax highlighting that contribute to package size.

## Example Migration

**Before (Taskfile):**
```yaml
version: '3'

vars:
  commit:
    sh: git rev-parse --short HEAD

tasks:
  build:
    cmds:
      - echo "Building {{.commit}}"
      - go build -ldflags "-X main.Commit={{.commit}}"
```

**After (Atkins):**
```yaml
vars:
  commit: $(git rev-parse --short HEAD)

tasks:
  build:
    cmds:
      - echo "Building ${{ commit }}"
      - go build -ldflags "-X main.Commit=${{ commit }}"
```
