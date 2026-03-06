# Structure

This is the table of contents for the docs:

- Getting Started
  - [Introduction](./getting-started/introduction.md)
  - [Installation](./getting-started/installation.md)
- Usage
  - [Configuration](./usage/configuration.md)
  - [Pipelines, Jobs and Steps](./usage/pipelines-jobs-steps.md)
  - [Skills](./usage/skills.md)
  - [CLI Flags](./usage/cli-flags.md)
  - [Job Targeting](./usage/job-targeting.md)
  - [Script Mode](./usage/script-mode.md)
  - [Automation (JSON/YAML)](./usage/automation.md)
- [Why use Atkins?](./why-atkins.md)
- Migrating
  - [Migrating to Atkins](./getting-started/migrating.md)
  - [Migration from Taskfile](./getting-started/migration-from-task.md)
  - [Migration from GitHub Actions](./getting-started/migration-from-github-actions.md)

## Getting Started

### Introduction

Overview of Atkins: what it is, key features (interactive tree display, parallel execution, multiple syntax styles, smart interpolation, skills system), a quick example, design philosophy, and output formats.

### Installation

Installation methods: from source with Go, binary release download, and Docker image. Includes verification steps and shebang support for executable pipelines.

## Usage

### Configuration

Pipeline configuration format and syntax. Covers both syntax flavors (Taskfile-style and GHA-style), variable interpolation (`${{ expr }}` and `$(command)`), environment inheritance, `vars:` block, `env:` block, `include:` for composition, and `when:` for conditional skill activation.

### Pipelines, Jobs and Steps

The three-tier execution hierarchy. Complete field reference for pipelines, jobs, and steps. Covers dependencies, detached jobs, string shorthand, deferred steps, for loops with task invocation, and conditional execution with `if:` (expr-lang syntax, available variables, operators, truthiness rules).

### Skills

Modular pipeline components. Covers skill locations (project and global), conditional activation with `when:`, namespacing, aliases, default jobs, cross-skill references, skill variables, and example skills (Go, Docker, Node.js).

### CLI Flags

Command-line options reference. Covers all flags (`--file`, `--list`, `--lint`, `--json`, `--yaml`, `--final`, `--log`, `--debug`, `--working-directory`, `--jail`), file discovery order, running and listing jobs, output modes, and stdin input.

### Job Targeting

Job resolution and targeting syntax. Covers basic targeting, namespaced jobs, root job targeting (`:` prefix), cross-pipeline task references, aliases, resolution order, and fuzzy matching.

### Script Mode

Executable pipelines and stdin input. Covers shebang execution, piping via stdin, positional arguments, and combining with CLI flags.

### Automation (JSON/YAML)

Machine-readable output for tooling integration. Covers list and execution output in JSON/YAML formats, schema, and use cases (LLM integration, CI discovery, script integration, monitoring).

## Why use Atkins?

When and why to choose Atkins over other solutions. Includes a comparison table with GitHub Actions, Taskfile, and Lefthook covering features like distributed execution, interpolation format, secrets management, environment inheritance, parallel execution, and more.

## Migrating

### Migrating to Atkins

Overview page for migration. Covers why to migrate (cleaner syntax, environment inheritance, local/CI parity, smaller binary, skills). Links to specific migration guides. Explains how to use Atkins in CI environments.

### Migration from Taskfile

Side-by-side syntax comparison between Taskfile and Atkins. Covers structure differences, shell substitution (`sh:` vs `$(...)`), template variables (`{{.Var}}` vs `${{ var }}`), environment handling, and what works directly without changes.

### Migration from GitHub Actions

Syntax mapping from GitHub Actions to Atkins. Covers triggers (not supported), runner selection (not supported), `uses:` actions (replaced by commands), dependencies (`needs:` vs `depends_on:`), variables, matrix builds (mapped to `for:` loops), conditional execution, and parallel execution.
