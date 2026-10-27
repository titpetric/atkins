# Structure

This is the table of contents for the docs:

- Getting Started
  - [Introduction](./getting-started/introduction.md)
  - [Installation](./getting-started/installation.md)
  - [Why use Atkins?](./getting-started/why-atkins.md)
- Reference
  - [Schema](./reference/schema.md)
  - [Pipeline](./reference/pipeline.md)
  - [Jobs](./reference/jobs.md)
  - [Steps](./reference/steps.md)
  - [Variables](./reference/variables.md)
  - [Includes](./reference/includes.md)
  - [Templating](./reference/templating.md)
- Usage
  - [Configuration](./usage/configuration.md)
  - [Pipelines](./usage/pipelines.md)
  - [Jobs](./usage/jobs.md)
  - [Steps](./usage/steps.md)
  - [Conditionals](./usage/conditionals.md)
  - [Loops](./usage/loops.md)
  - [Skills](./usage/skills.md)
  - [CLI Flags](./usage/cli-flags.md)
  - [Job Targeting](./usage/job-targeting.md)
  - [Script Mode](./usage/script-mode.md)
  - [Automation (JSON/YAML)](./usage/automation.md)
  - [Statistics](./usage/stats.md)
- Migrating
  - [Migrating to Atkins](./migrating/migrating.md)
  - [Migration from Taskfile](./migrating/migration-from-task.md)
  - [Migration from GitHub Actions](./migrating/migration-from-github-actions.md)

## Getting Started

### Introduction

Overview of Atkins: a definition, key features (interactive tree display, parallel execution, two syntax styles, smart interpolation, skills system), a quick example, design philosophy, and output formats.

### Installation

Installation methods: from source with Go, binary release download, and Docker image. Includes verification steps and shebang support for executable pipelines.

### Why use Atkins?

When and why to choose Atkins over other solutions. Includes a comparison table with GitHub Actions, Taskfile, and Lefthook covering features like distributed execution, interpolation format, secrets management, environment inheritance, parallel execution, and more.

## Reference

### Schema

The atkins.yml schema in one page: the keys a pipeline, a job and a step accept, and the GitHub Actions / Taskfile spelling each pairs with.

### Pipeline

Pipeline-level fields: `name`, `dir`, `vars`, `env`, `jobs`/`tasks`, `include`, `when`.

### Jobs

Job-level fields: `desc`, `steps`/`cmds`, dependencies, conditions, loops, and the run-behavior flags (`detach`, `passthru`, `interactive`, `summarize`, `quiet`).

### Steps

Step-level fields: `run`/`cmd`/`cmds`/`task`, conditions, loops, `deferred`, and the same run-behavior flags a job carries.

### Variables

`${{ name }}` interpolation, `$(command)` substitution, `${NAME}` environment expansion, and variable scoping across pipeline, job, step and loop.

### Includes

Composing a pipeline from two or more YAML files with `include:` at the pipeline, job or step level.

### Templating

The [expr-lang](https://expr-lang.org/) expression syntax of `if:` conditions.

## Usage

### Configuration

Pipeline configuration format and syntax. Covers both syntax flavors (Taskfile-style and GHA-style), variable interpolation (`${{ expr }}` and `$(command)`), environment inheritance, `vars:` block, `env:` block, `include:` for composition, and `when:` for conditional skill activation.

### Pipelines

Top-level pipeline configuration. Covers pipeline fields (name, dir, vars, env, jobs/tasks).

### Jobs

Job configuration and dependencies. Covers job fields, dependencies, detached jobs, conditional execution, and string shorthand.

### Steps

Step configuration and execution. Covers step fields, task invocation, deferred steps, and for loops.

### Conditionals

Conditional execution with if expressions. Covers expr-lang syntax, available variables, operators, truthiness rules, and skipped output.

### Loops

For loops and iteration. Covers loop syntax, required variables, and task invocation with loop variables.

### Skills

Modular pipeline components. Covers skill locations (project and global), conditional activation with `when:` and glob patterns, namespacing, aliases, default jobs, cross-skill references, skill variables, example skills (Go, Docker, Node.js), and vendoring global skills into a repository with `--vendor` / `--write`.

### CLI Flags

Command-line options reference: file discovery order, running and listing jobs, output modes, and stdin input.

**Choosing what runs**

| Flag                  | Short | Description                                                                    |
|-----------------------|-------|--------------------------------------------------------------------------------|
| `--file`              | `-f`  | Pipeline file to use, instead of the discovered one                            |
| `--working-directory` | `-w`  | Change to this directory before running                                        |
| `--jail`              |       | Load skills from the project only, ignoring parent folders and `$HOME/.atkins` |
| `--list`              | `-l`  | List the jobs and dependencies instead of running them                         |
| `--lint`              |       | Check the pipeline for errors and exit                                         |

**Output**

| Flag        | Short | Description                                             |
|-------------|-------|---------------------------------------------------------|
| `--json`    | `-j`  | Emit the run, or the listing, as JSON                   |
| `--yaml`    | `-y`  | Emit the run, or the listing, as YAML                   |
| `--final`   |       | Print the tree once at the end rather than redrawing it |
| `--log`     |       | Write an execution log to this file                     |
| `--debug`   |       | Print interpolation, evaluation and timing detail       |
| `--version` | `-v`  | Print the version and build information                 |

**Statistics**

| Flag      | Description                                                               |
|-----------|---------------------------------------------------------------------------|
| `--stats` | Report the recorded commands by the time they have taken, and run nothing |
| `--local` | Run without recording anything in the event database                      |

**Project setup**

| Flag       | Description                                                        |
|------------|--------------------------------------------------------------------|
| `--config` | Open the configuration menu over `.atkins/config.yml`              |
| `--vendor` | Report the skills this repository uses from `$HOME/.atkins/skills` |
| `--write`  | With `--vendor`, write them into `.atkins/skills`                  |

### Job Targeting

Job resolution and targeting syntax. Covers basic targeting, namespaced jobs, root job targeting (`:` prefix), cross-pipeline task references, aliases, resolution order, and fuzzy matching.

### Script Mode

Executable pipelines and stdin input. Covers shebang execution, piping via stdin, positional arguments, and combining with CLI flags.

### Automation (JSON/YAML)

Machine-readable output for tooling integration. Covers list and execution output in JSON/YAML formats, schema, and use cases (LLM integration, CI discovery, script integration, monitoring).

### Statistics

The event database and what it answers. Covers where the file lives and how atkins finds it, the `run` and `command` tables and the question each answers, the normalised command the statistics group by and the rules that produce it, the three parts of the `atkins --stats` report, the JSON and YAML a program reads it from, turning recording off with `--local` and `storage.record`, the 31 day window, and reading the file from another program.

## Migrating

### Migrating to Atkins

Overview page for migration. Covers why to migrate (cleaner syntax, environment inheritance, local/CI parity, smaller binary, skills). Links to specific migration guides. Explains how to use Atkins in CI environments.

### Migration from Taskfile

Side-by-side syntax comparison between Taskfile and Atkins. Covers structure differences, shell substitution (`sh:` vs `$(...)`), template variables (`{{.Var}}` vs `${{ var }}`), environment handling, and what works directly without changes.

### Migration from GitHub Actions

Syntax mapping from GitHub Actions to Atkins. Covers triggers (not supported), runner selection (not supported), `uses:` actions (replaced by commands), dependencies (`needs:` vs `depends_on:`), variables, matrix builds (mapped to `for:` loops), conditional execution, and parallel execution.
