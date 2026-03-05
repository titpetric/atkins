---
title: Introduction
subtitle: Portable command runner and CI tooling
layout: page
---

# Atkins

Atkins is a minimal command runner focused on local development and CI/CD environments. It features an interactive CLI status tree showing job progress, with support for parallel execution of jobs and steps.

## Key Features

- **Interactive tree display** - See which jobs and steps are running in real-time
- **Parallel execution** - Run jobs and steps concurrently with `detach: true`
- **Multiple syntax styles** - Supports both Taskfile-style (`tasks/cmds`) and GitHub Actions-style (`jobs/steps`)
- **Smart interpolation** - Use `${{ var }}` for variables and `$(command)` for shell substitution
- **Cross-pipeline references** - Reference tasks from other pipelines with `:skill:task` syntax
- **Skills system** - Modular, reusable pipeline components

## Quick Example

Create an `atkins.yml` file:

```yaml
name: My Project

vars:
  greeting: Hello

tasks:
  default:
    desc: Run the greeting
    steps:
      - run: echo "${{ greeting }}, World!"

  build:
    desc: Build the project
    steps:
      - run: go build ./...
```

Run it:

```bash
# Run the default task
atkins

# List available tasks
atkins -l

# Run a specific task
atkins build
```

## Design Philosophy

Atkins was designed to address common pain points with existing task runners:

1. **YAML-friendly syntax** - Variable interpolation uses `${{ }}` which doesn't conflict with YAML parsing
2. **Environment inheritance** - Commands inherit the full shell environment without explicit declarations
3. **Minimal dependencies** - Small binary size without unnecessary features
4. **Familiar patterns** - Borrows concepts from Taskfile, GitHub Actions, and Drone CI

## Output Formats

Atkins supports multiple output formats:

```bash
# Interactive tree (default)
atkins

# List tasks as YAML (for LLM/tooling integration)
atkins -l -y

# List tasks as JSON
atkins -l -j

# Final tree only (no live updates)
atkins --final
```
