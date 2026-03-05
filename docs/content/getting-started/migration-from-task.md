---
title: Migration from Task
subtitle: Moving from Taskfile to Atkins
layout: page
---

# Migrating from Taskfiles

Atkins supports a Taskfile-compatible structure, making migration straightforward for simple pipelines.

## Structure Comparison

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

The main difference: Atkins doesn't require a `version` field.

## Shell Substitution

**Taskfile** uses the `sh:` field:
```yaml
vars:
  uname:
    sh: uname -n
```

**Atkins** uses bash-style `$(...)`:
```yaml
vars:
  uname: $(uname -n)
```

The `$(...)` syntax can also be used inline within commands.

## Variable Interpolation

**Taskfile** uses Go templates:
```yaml
vars:
  foo: bar
  bar: "{{.foo}}"  # Must be quoted

tasks:
  default:
    cmds:
      - echo "{{.foo}}"
```

**Atkins** uses `${{ }}` syntax:
```yaml
vars:
  foo: bar
  bar: ${{foo}}  # No quotes needed

tasks:
  default:
    cmds:
      - echo "${{foo}}"
```

Benefits:
- No quoting required in YAML
- No `.` prefix for variable names
- Doesn't conflict with bash `${var}` syntax

## Environment Variables

**Taskfile** requires explicit environment declarations and doesn't inherit from the shell by default.

**Atkins** inherits the full environment automatically. You can still declare environment variables at any level:

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

Both support shorthand syntax for simple commands:

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
atkins -l  # Shows all tasks with commands if no description
```

Atkins always shows tasks, using the command as the description if none is provided.

## What Works Directly

Simple Taskfiles without interpolation work identically:

```yaml
tasks:
  up: docker compose up -d --remove-orphans
  down: docker compose down --remove-orphans
```

## What Needs Changes

1. **Shell substitution**: `sh:` → `$(...)`
2. **Variable interpolation**: `{{.var}}` → `${{var}}`
3. **Quoted values**: Often can remove quotes around interpolated values
