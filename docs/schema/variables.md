---
title: Variables & Interpolation
subtitle: Variable declaration and interpolation syntax
layout: page
---

# Variables & Interpolation

Atkins provides powerful variable interpolation that's YAML-friendly and doesn't conflict with shell syntax.

## Declaring Variables

### Pipeline Level

```yaml
vars:
  version: 1.0.0
  app_name: myapp

jobs:
  build:
    steps:
      - run: echo "Building ${{ app_name }} v${{ version }}"
```

### Job Level

```yaml
jobs:
  build:
    vars:
      output_dir: ./bin
    steps:
      - run: go build -o ${{ output_dir }}/app
```

### Step Level

```yaml
steps:
  - vars:
      timestamp: $(date +%s)
    run: echo "Build at ${{ timestamp }}"
```

## Interpolation Syntax

### Variable Reference: `${{ }}`

```yaml
vars:
  name: world

steps:
  - run: echo "Hello, ${{ name }}!"
```

### Shell Substitution: `$(...)`

Execute shell commands inline:

```yaml
vars:
  git_commit: $(git rev-parse --short HEAD)
  current_date: $(date +%Y-%m-%d)

steps:
  - run: echo "Commit ${{ git_commit }} on ${{ current_date }}"
```

### Combining Both

```yaml
vars:
  branch: $(git branch --show-current)
  deploy_target: ${{ branch == "main" ? "prod" : "staging" }}
```

## Why This Syntax?

| Syntax | Purpose | Example |
|--------|---------|---------|
| `${{ var }}` | Atkins variables | `${{ version }}` |
| `$(cmd)` | Shell substitution | `$(date +%s)` |
| `${VAR}` | Bash variables | `${HOME}` |
| `$VAR` | Bash variables | `$PATH` |

This design ensures:
- No YAML quoting required for `${{ }}`
- No conflict with bash `${var}` or `$var`
- Shell substitution works naturally

## Variable Types

### Strings

```yaml
vars:
  message: "Hello, World"
  path: /usr/local/bin
```

### Numbers

```yaml
vars:
  port: 8080
  timeout: 30
```

### Lists

```yaml
vars:
  services:
    - api
    - web
    - worker
```

Use with for loops:

```yaml
steps:
  - for: service in services
    run: docker restart ${{ service }}
```

### Maps

```yaml
vars:
  config:
    host: localhost
    port: 5432

steps:
  - run: psql -h ${{ config.host }} -p ${{ config.port }}
```

## Scope and Precedence

Variables cascade from outer to inner scope:

1. Pipeline vars (lowest priority)
2. Job vars
3. Step vars
4. Loop vars (highest priority)

```yaml
vars:
  message: "pipeline"

jobs:
  test:
    vars:
      message: "job"  # Overrides pipeline
    steps:
      - vars:
          message: "step"  # Overrides job
        run: echo ${{ message }}  # Outputs: step
```

## Include Files

Load variables from external files:

```yaml
include:
  - config/defaults.yml
  - config/secrets.yml
```

**config/defaults.yml:**
```yaml
app_name: myapp
version: 1.0.0
```

Variables from included files are available in the pipeline.

## Environment Variables

### Setting Environment

```yaml
env:
  vars:
    GOOS: linux
    GOARCH: amd64
```

### Using Environment in Commands

Environment variables are available to commands directly:

```yaml
env:
  vars:
    API_KEY: secret123

steps:
  - run: curl -H "Authorization: $API_KEY" https://api.example.com
```

### Environment from Variables

```yaml
vars:
  go_os: linux

env:
  vars:
    GOOS: ${{ go_os }}
```

## Expressions

Atkins uses [expr-lang](https://expr-lang.org/) for expressions:

### Conditionals

```yaml
vars:
  env: production
  debug: ${{ env != "production" }}
```

### Ternary

```yaml
vars:
  branch: $(git branch --show-current)
  deploy_env: ${{ branch == "main" ? "prod" : "dev" }}
```

### String Operations

```yaml
vars:
  filename: myapp-linux-amd64
  basename: ${{ split(filename, "-")[0] }}
```

## Built-in Variables

Some variables are automatically available:

| Variable | Description |
|----------|-------------|
| `item` | Current item in a for loop |
| `index` | Current index in a for loop |

## Examples

### Dynamic Version

```yaml
vars:
  version: $(git describe --tags --always)
  commit: $(git rev-parse --short HEAD)
  build_time: $(date -u +%Y-%m-%dT%H:%M:%SZ)

jobs:
  build:
    steps:
      - run: |
          go build -ldflags="\
            -X main.Version=${{ version }} \
            -X main.Commit=${{ commit }} \
            -X main.BuildTime=${{ build_time }}"
```

### Configuration Loading

```yaml
vars:
  config: $(cat config.json)
  api_url: ${{ config.api_url }}
  timeout: ${{ config.timeout }}
```

### Conditional Deployment

```yaml
vars:
  branch: $(git branch --show-current)
  is_release: ${{ startsWith(branch, "release/") }}
  deploy_target: ${{ is_release ? "production" : "staging" }}

jobs:
  deploy:
    if: branch == "main" || is_release
    steps:
      - run: deploy --target ${{ deploy_target }}
```
