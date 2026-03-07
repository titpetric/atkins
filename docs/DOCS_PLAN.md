# Atkins Documentation Restructure Plan

## Overview

This plan restructures the atkins documentation to:
1. Align with Taskfile reference structure where applicable (without versioning)
2. Use vuego-cli docs embedding syntax for live examples
3. Create `article-name/` folders with testable examples matching each `article-name.md`
4. Ensure docs stay current through testable, executable examples

---

## vuego-cli Docs Embedding Syntax

The documentation will use these directives to embed examples:

### `@file` - Display file content as code
```markdown
@file "Label" filename.ext
```

### `@tabs` - Group multiple views
```markdown
@tabs
@file "Pipeline" example.yml
@file "Output" output.txt

```

### `@example` - Show preview + code (for vuego templates)
```markdown
@example component.vuego
```

---

## Proposed Directory Structure

```
docs/content/
├── README.md                          # Landing page
├── STRUCTURE.md                       # Table of contents
├── data/
│   ├── menu.yml                       # Navigation menu
│   └── pkg.yml                        # Package metadata
├── theme.yml                          # Theme configuration
│
├── getting-started/
│   ├── introduction.md
│   ├── introduction/
│   │   ├── hello-world.yml            # Minimal pipeline
│   │   └── quick-start.yml            # Feature showcase
│   ├── installation.md
│   └── why-atkins.md
│
├── reference/                         # NEW: Schema reference (like Taskfile)
│   ├── schema.md                      # Pipeline schema overview
│   ├── schema/
│   │   ├── minimal.yml
│   │   ├── full-example.yml
│   │   └── output.txt
│   ├── pipeline.md                    # Pipeline-level fields
│   ├── pipeline/
│   │   ├── basic.yml
│   │   ├── with-env.yml
│   │   └── with-vars.yml
│   ├── jobs.md                        # Job schema reference
│   ├── jobs/
│   │   ├── basic.yml
│   │   ├── with-deps.yml
│   │   ├── detached.yml
│   │   └── conditional.yml
│   ├── steps.md                       # Step schema reference
│   ├── steps/
│   │   ├── run-cmd.yml
│   │   ├── task-invoke.yml
│   │   ├── deferred.yml
│   │   ├── for-loop.yml
│   │   └── conditional.yml
│   ├── variables.md                   # Variable interpolation
│   ├── variables/
│   │   ├── static.yml
│   │   ├── dynamic-shell.yml
│   │   ├── nested.yml
│   │   └── env-vars.yml
│   ├── includes.md                    # Include files
│   ├── includes/
│   │   ├── main.yml
│   │   ├── ci/
│   │   │   ├── build.yml
│   │   │   └── test.yml
│   │   └── output.txt
│   └── templating.md                  # Expression syntax
│       └── templating/
│           ├── expr-examples.yml
│           └── functions.yml
│
├── usage/                             # Guides (reorganized)
│   ├── configuration.md               # Overview, syntax flavors
│   ├── configuration/
│   │   ├── taskfile-style.yml
│   │   ├── gha-style.yml
│   │   └── mixed-style.yml
│   ├── pipelines.md
│   ├── pipelines/
│   │   ├── basic.yml
│   │   └── complete.yml
│   ├── jobs.md
│   ├── jobs/
│   │   ├── dependencies.yml
│   │   ├── detached.yml
│   │   └── string-shorthand.yml
│   ├── steps.md
│   ├── steps/
│   │   ├── basic.yml
│   │   ├── task-ref.yml
│   │   ├── deferred.yml
│   │   └── for-loop.yml
│   ├── conditionals.md
│   ├── conditionals/
│   │   ├── job-if.yml
│   │   ├── step-if.yml
│   │   └── complex-expr.yml
│   ├── loops.md
│   ├── loops/
│   │   ├── list-iteration.yml
│   │   ├── nested-vars.yml
│   │   └── matrix-like.yml
│   ├── skills.md
│   ├── skills/
│   │   ├── example-skill.yml
│   │   ├── when-activation.yml
│   │   └── cross-reference.yml
│   ├── cli-flags.md
│   ├── job-targeting.md
│   ├── job-targeting/
│   │   ├── namespace.yml
│   │   ├── root-target.yml
│   │   └── aliases.yml
│   ├── script-mode.md
│   ├── script-mode/
│   │   ├── shebang.yml
│   │   └── stdin-example.yml
│   └── automation.md
│       └── automation/
│           ├── pipeline.yml
│           ├── json-output.json
│           └── yaml-output.yaml
│
└── migrating/
    ├── migrating.md
    ├── migration-from-task.md
    ├── migration-from-task/
    │   ├── taskfile-before.yml        # Original Taskfile
    │   └── atkins-after.yml           # Converted Atkins
    ├── migration-from-github-actions.md
    └── migration-from-github-actions/
        ├── workflow-before.yml        # Original GHA workflow
        └── atkins-after.yml           # Converted Atkins
```

---

## Reference Section Schema (Aligned with Taskfile)

### `reference/schema.md` - Pipeline Schema Overview

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Pipeline name |
| `dir` | string | Working directory |
| `vars` | map | Pipeline variables |
| `env` | object | Environment variables |
| `jobs` / `tasks` | map | Job/task definitions |
| `include` | string/list | External file inclusion |
| `when` | object | Skill activation conditions |

### `reference/pipeline.md` - Pipeline Properties

Covers: `name`, `dir`, `vars`, `env`, `include`, `when`

### `reference/jobs.md` - Job Properties

| Field | Type | Description |
|-------|------|-------------|
| `desc` | string | Short description |
| `depends_on` / `deps` | list | Job dependencies |
| `steps` / `cmds` | list | Steps to execute |
| `vars` | map | Job-level variables |
| `env` | object | Job-level environment |
| `if` | string | Conditional execution |
| `dir` | string | Working directory |
| `detach` | bool | Run in background |
| `show` | bool | Visibility in list |

### `reference/steps.md` - Step Properties

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Step name |
| `run` / `cmd` / `cmds` | string/list | Commands to execute |
| `task` | string | Task to invoke |
| `if` | string | Conditional execution |
| `for` | string | Loop iteration |
| `requires` | list | Required loop variables |
| `dir` | string | Working directory |
| `env` | object | Step environment |
| `timeout` | string | Execution timeout |
| `deferred` | bool | Run on cleanup |
| `detach` | bool | Run in background |
| `tty` | bool | Allocate PTY |
| `interactive` | bool | Live streaming |
| `verbose` | bool | Show output |
| `quiet` | bool | Suppress output |
| `passthru` | bool | Pass through output |
| `summarize` | bool | Show summary |

### `reference/variables.md` - Variable System

- Static values
- Dynamic shell execution `$(command)`
- Interpolation syntax `${{ expr }}`
- Nested access `${{ build.goarch }}`
- Environment variables

### `reference/includes.md` - File Inclusion

- Glob patterns: `include: ci/*.yml`
- Multiple files
- Namespace behavior

### `reference/templating.md` - Expression Syntax

- expr-lang expressions
- Available functions
- Boolean operators
- Type coercion

---

## Example Article with Embedded Examples

### `reference/jobs.md` with examples

```markdown
---
title: Jobs
subtitle: Job schema reference
layout: page
---

Jobs define units of work in a pipeline. Each job contains steps to execute.

## Basic Job

@tabs
@file "Pipeline" jobs/basic.yml
@file "Output" jobs/basic-output.txt


## Job with Dependencies

Jobs can depend on other jobs using `depends_on`:

@tabs
@file "Pipeline" jobs/with-deps.yml


## Detached Jobs

Run jobs in the background with `detach: true`:

@tabs
@file "Pipeline" jobs/detached.yml


## Schema Reference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `desc` | string | - | Short description shown in `--list` |
| `depends_on` | list | [] | Jobs to run before this job |
| `steps` | list | [] | Steps to execute |
...
```

### Example File: `reference/jobs/basic.yml`

```yaml
name: Basic Job Example

jobs:
  build:
    desc: Build the application
    steps:
      - run: echo "Building..."
      - run: go build ./...
```

### Example File: `reference/jobs/with-deps.yml`

```yaml
name: Job Dependencies

jobs:
  default:
    desc: Run everything
    depends_on: [lint, test]
    steps:
      - run: echo "All checks passed"

  lint:
    steps:
      - run: golangci-lint run

  test:
    steps:
      - run: go test ./...
```

---

## Menu Structure Update

```yaml
# data/menu.yml
menu:
  - type: group
    label: Getting Started
    items:
      - label: Introduction
        url: /getting-started/introduction
      - label: Installation
        url: /getting-started/installation
      - label: Why use Atkins?
        url: /getting-started/why-atkins

  - type: group
    label: Reference
    items:
      - label: Schema Overview
        url: /reference/schema
      - label: Pipeline
        url: /reference/pipeline
      - label: Jobs
        url: /reference/jobs
      - label: Steps
        url: /reference/steps
      - label: Variables
        url: /reference/variables
      - label: Includes
        url: /reference/includes
      - label: Templating
        url: /reference/templating

  - type: group
    label: Usage Guide
    items:
      - label: Configuration
        url: /usage/configuration
      - label: Pipelines
        url: /usage/pipelines
      - label: Jobs
        url: /usage/jobs
      - label: Steps
        url: /usage/steps
      - label: Conditionals
        url: /usage/conditionals
      - label: Loops
        url: /usage/loops
      - label: Skills
        url: /usage/skills
      - label: CLI Flags
        url: /usage/cli-flags
      - label: Job Targeting
        url: /usage/job-targeting
      - label: Script Mode
        url: /usage/script-mode
      - label: Automation
        url: /usage/automation

  - type: group
    label: Migrating
    items:
      - label: Overview
        url: /migrating/migrating
      - label: From Taskfile
        url: /migrating/migration-from-task
      - label: From GitHub Actions
        url: /migrating/migration-from-github-actions
```

---

## Maintenance Intent

### Purpose of Example Folders

Each `article-name/` folder serves as:

1. **Living Documentation**: Examples are actual runnable pipelines
2. **Testable Fixtures**: Can be executed via `atkins -f example.yml` in CI
3. **Embedded Content**: Pulled into docs via `@file` directive
4. **Single Source of Truth**: Code shown in docs matches tested code

### CI Integration

Add a docs validation job to `.atkins.yml`:

```yaml
jobs:
  test:docs:
    desc: Validate documentation examples
    steps:
      - name: Lint examples
        for: example in $(find docs/content -name "*.yml" -path "*/*/")
        run: atkins --lint -f ${{ example }}

      - name: Run example pipelines
        for: example in $(find docs/content -name "*.yml" -path "*/*/" -name "*.yml")
        if: '!contains(example, "before") && !contains(example, "taskfile")'
        run: atkins -f ${{ example }}
```

### Update Workflow

When updating features:

1. Update the example YAML in `article-name/`
2. Run the example to capture new output
3. Save output to `article-name/output.txt` if needed
4. The markdown article automatically shows current content via `@file`

### Output Capture Pattern

For examples showing output, capture and commit:

```bash
atkins -f docs/content/usage/jobs/dependencies.yml > docs/content/usage/jobs/dependencies-output.txt
```

Then embed in docs:

```markdown
@tabs
@file "Pipeline" jobs/dependencies.yml
@file "Output" jobs/dependencies-output.txt

```

---

## Key Differences from Taskfile Docs

| Aspect | Taskfile | Atkins |
|--------|----------|--------|
| Versioning | `version: '3'` required | No versioning |
| Structure keywords | `tasks:` only | `jobs:` or `tasks:` |
| Step keywords | `cmds:` only | `steps:`, `cmds:`, `run:`, `cmd:` |
| Variable syntax | `{{.Var}}` (Go template) | `${{ var }}` |
| Shell execution | `sh: command` | `$(command)` |
| Environment | Explicit declaration | Full inheritance |
| Include syntax | `includes:` map | `include:` glob |
| Conditional | Task-level only | Job and step level |
| Skills | N/A | Built-in modular system |

---

## Implementation Priority

### Phase 1: Reference Section (New)
1. Create `reference/` directory
2. Write `schema.md` overview
3. Create `pipeline.md`, `jobs.md`, `steps.md`, `variables.md`
4. Add example folders with runnable YAML files

### Phase 2: Update Usage Section
1. Add example folders to existing articles
2. Convert inline code blocks to `@file` embeds
3. Add output captures where helpful

### Phase 3: Migrating Section
1. Add before/after example folders
2. Use `@tabs` for side-by-side comparison

### Phase 4: CI Validation
1. Add docs example linting to CI
2. Add example execution tests
3. Set up output capture automation

---

## Rendering

Render documentation with:

```bash
vuego-cli docs ./docs/content
```

The server will:
- Serve markdown with embedded examples
- Process `@file`, `@tabs`, `@example` directives
- Apply theme from `theme.yml`
- Generate navigation from `data/menu.yml`
