---
title: Pipelines, Jobs and Steps
subtitle: Understanding the execution hierarchy
layout: page
---

# Pipelines, Jobs and Steps

Atkins uses a three-tier execution hierarchy: pipelines contain jobs, and jobs contain steps. Understanding this structure helps you organize your tasks effectively and use features like dependencies, parallel execution, and cross-job references.

```
Pipeline (atkins.yml)
├─ Job: build
│  ├─ Step: run tests
│  └─ Step: compile binary
├─ Job: deploy
│  ├─ Step: push image
│  └─ Step: apply config
```

## Pipeline

The pipeline is the top-level structure defined in your `atkins.yml` file. It sets the context for all jobs—name, working directory, variables, and environment.

```yaml
name: My Project
dir: ./src

vars:
  version: 1.0.0
  app_name: myapp

env:
  vars:
    GO111MODULE: "on"

jobs:
  build:
    steps:
      - run: go build -o ${{ app_name }}
  test:
    steps:
      - run: go test ./...
```

### Pipeline Fields

| Field | Description |
|-------|-------------|
| `name:` | Pipeline display name |
| `dir:` | Working directory for the pipeline |
| `vars:` | Pipeline-level variables available to all jobs |
| `env:` | Pipeline-level environment variables |
| `jobs:` | Job definitions (map of name → job) |
| `tasks:` | Alias for `jobs:` (Taskfile-style) |

`jobs:` and `tasks:` are interchangeable—use whichever style you prefer.

## Jobs

A job groups related steps and controls how they execute. Jobs are defined as a map under `jobs:` or `tasks:`.

```yaml
tasks:
  lint:
    desc: Run linters
    steps:
      - run: golangci-lint run

  build:
    desc: Build the application
    depends_on: lint
    timeout: 10m
    steps:
      - run: go build -o app .

  test:
    desc: Run test suite
    depends_on: lint
    tty: true
    steps:
      - run: gotestsum ./...
```

### Job Fields

| Field | Description |
|-------|-------------|
| `desc:` | Description shown in job listings |
| `steps:` | List of steps to execute |
| `cmds:` | Alias for `steps:` (Taskfile-style) |
| `cmd:` | Single command shorthand |
| `run:` | Alias for `cmd:` |
| `depends_on:` | Job dependencies—string or list of job names |
| `detach: true` | Run the job in background (parallel) |
| `aliases:` | Alternative names for invoking the job |
| `requires:` | Required variables when invoked via a for loop |
| `if:` | Conditional execution ([expr-lang](https://expr-lang.org/) expression) |
| `dir:` | Working directory for all steps |
| `timeout:` | Maximum execution time (e.g. `"10m"`, `"300s"`) |
| `passthru: true` | Output printed with tree indentation |
| `tty: true` | Allocate a PTY for color output |
| `interactive: true` | Stream output live and connect stdin |
| `quiet: true` | Suppress output |
| `summarize: true` | Summarize output |
| `show:` | Control visibility in tree (`true`/`false`/omit) |
| `vars:` | Job-level variables |
| `env:` | Job-level environment variables |

### Dependencies

Use `depends_on:` to declare that a job must run after another:

```yaml
jobs:
  lint:
    steps:
      - run: golangci-lint run

  build:
    depends_on: lint
    steps:
      - run: go build .

  deploy:
    depends_on: [build, test]
    steps:
      - run: ./deploy.sh
```

### Detached (Background) Jobs

Jobs with `detach: true` run in the background, allowing subsequent jobs to start immediately:

```yaml
jobs:
  server:
    detach: true
    steps:
      - run: docker compose up

  test:
    depends_on: server
    steps:
      - run: go test ./...
```

### Conditional Execution

Jobs can be conditionally executed using `if:` with an [expr-lang](https://expr-lang.org/) expression:

```yaml
jobs:
  deploy:
    if: branch == "main"
    steps:
      - run: ./deploy.sh
```

### String Shorthand

For simple single-command jobs, use string shorthand:

```yaml
tasks:
  up: docker compose up -d
  down: docker compose down
  logs: docker compose logs -f
```

This is equivalent to:

```yaml
tasks:
  up:
    steps:
      - run: docker compose up -d
```

## Steps

Steps are the individual commands or actions within a job. They execute sequentially by default.

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

### Step Fields

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

### Simple String Steps

Steps can be plain strings:

```yaml
steps:
  - echo "Hello"
  - go build .
  - go test ./...
```

### Multiple Commands

Use `cmds:` to run several commands in a single step:

```yaml
steps:
  - name: Cleanup
    cmds:
      - rm -rf dist/
      - mkdir -p dist/
      - cp README.md dist/
```

### Task Invocation

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

  test:
    steps:
      - run: go test ./...
```

### Deferred Steps

Deferred steps run after all other steps complete, regardless of success or failure. Useful for cleanup:

```yaml
steps:
  - defer:
      run: docker compose down
  - run: docker compose up -d
  - run: go test ./...
```

Or using the explicit `deferred: true` flag:

```yaml
steps:
  - run: docker compose down
    deferred: true
  - run: docker compose up -d
  - run: go test ./...
```

### Working Directory

```yaml
steps:
  - name: Build frontend
    dir: ./frontend
    run: npm run build
  - name: Build backend
    dir: ./backend
    run: go build .
```

## For Loops

Invoke tasks repeatedly with different loop variables:

```yaml
vars:
  components:
    - src/main
    - src/utils

tasks:
  build:
    steps:
      - for: component in components
        task: build_component

  build_component:
    requires: [component]
    steps:
      - run: make build -C "${{ component }}"
```

### How It Works

1. **`for:` in a step** — defines the loop with `for: variable in collection`
2. **`task:` in the same step** — names the task to invoke for each iteration
3. **Loop variable** — becomes available in the invoked task as `${{ variable }}`
4. **`requires:`** — the invoked task can declare required variables to validate they are present

### Advanced Example

```yaml
vars:
  environments:
    - dev
    - staging
    - prod
  service_version: 1.2.3

tasks:
  deploy_all:
    desc: Deploy to all environments
    steps:
      - for: env in environments
        task: deploy_service

  deploy_service:
    desc: Deploy to a specific environment
    requires: [env, service_version]
    steps:
      - run: kubectl apply -f config/${{ env }}/deployment.yml --image=app:${{ service_version }}
```

If a required variable is missing, execution fails with a clear error:

```
job 'deploy_service' requires variables [env service_version] but missing: [env]
```

## Conditional Execution

Jobs and steps can be conditionally executed using the `if:` field. Conditions are evaluated using [expr-lang](https://expr-lang.org/), a simple expression language.

### Basic Syntax

```yaml
jobs:
  deploy:
    if: environment == "production"
    steps:
      - run: ./deploy.sh

  notify:
    if: send_notifications == true
    steps:
      - run: ./notify.sh
```

Steps can also have conditions:

```yaml
steps:
  - name: Deploy to production
    if: environment == "production"
    run: ./deploy.sh
  - name: Deploy to staging
    if: environment == "staging"
    run: ./deploy-staging.sh
```

### Available Variables

Conditions have access to:

1. **Pipeline variables** — All variables defined in `vars:` blocks
2. **Environment variables** — All environment variables (from shell and `env:` blocks)
3. **Loop variables** — When inside a `for:` loop, the loop variable is available

```yaml
vars:
  deploy_env: production
  enable_tests: true

env:
  vars:
    CI: true

jobs:
  build:
    if: enable_tests == true
    steps:
      - run: go test ./...

  deploy:
    if: deploy_env == "production" && CI == "true"
    steps:
      - run: ./deploy.sh
```

### Expression Syntax

Expr-lang supports common operators and comparisons:

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equals | `env == "prod"` |
| `!=` | Not equals | `env != "dev"` |
| `&&` | Logical AND | `a == 1 && b == 2` |
| `\|\|` | Logical OR | `a == 1 \|\| b == 2` |
| `!` | Logical NOT | `!skip_tests` |
| `>`, `<`, `>=`, `<=` | Comparisons | `num_retries > 0` |
| `in` | Contains | `"prod" in environments` |
| `matches` | Regex match | `branch matches "^release/"` |

### Examples

**String comparisons:**
```yaml
if: environment == "production"
if: branch != "main"
```

**Boolean variables:**
```yaml
if: enable_deploy
if: !skip_tests
if: run_integration_tests == true
```

**Combining conditions:**
```yaml
if: environment == "production" && branch == "main"
if: skip_tests || environment == "development"
```

**Checking for values in lists:**
```yaml
vars:
  allowed_envs:
    - staging
    - production

jobs:
  deploy:
    if: environment in allowed_envs
    steps:
      - run: ./deploy.sh
```

**Pattern matching:**
```yaml
if: branch matches "^release/.*"
if: tag matches "^v[0-9]+\\.[0-9]+\\.[0-9]+$"
```

### Truthiness

Values are coerced to boolean as follows:

| Value | Result |
|-------|--------|
| `true` | true |
| `false` | false |
| `nil` / undefined | false |
| `""` (empty string) | false |
| `"false"`, `"0"` | false |
| Any other string | true |
| `0` | false |
| Any other number | true |

This means you can use variables directly as conditions:

```yaml
vars:
  deploy: true
  skip_tests: false

jobs:
  build:
    if: deploy        # true - runs
    steps:
      - run: ./build.sh

  test:
    if: skip_tests    # false - skipped
    steps:
      - run: ./test.sh
```

### Undefined Variables

Undefined variables evaluate to `nil` (falsy) rather than causing an error:

```yaml
jobs:
  optional:
    if: maybe_defined   # Skipped if maybe_defined is not set
    steps:
      - run: ./optional.sh
```

### Skipped Output

When a job or step is skipped due to a condition, the tree output shows the condition:

```
✓ build
⊘ deploy (if: environment == "production")
```

## Complete Example

```yaml
name: My App Pipeline

vars:
  app: myapp
  version: 1.0.0

tasks:
  default:
    desc: Build and test
    depends_on: [lint, test]

  lint:
    desc: Run linters
    steps:
      - run: golangci-lint run

  test:
    desc: Run tests
    tty: true
    steps:
      - run: gotestsum ./...

  build:
    desc: Build binary
    depends_on: lint
    timeout: 5m
    steps:
      - run: go build -o bin/${{ app }} .

  up: docker compose up -d
  down: docker compose down
```

## See Also

- [CLI Flags](./cli-flags) — Command-line options
- [Job Targeting](./job-targeting) — Running specific jobs
- [Skills](./skills) — Reusable pipeline components
