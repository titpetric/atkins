---
title: Migration from GitHub Actions
subtitle: Using Atkins with GHA-style syntax
layout: page
---

# Migration from GitHub Actions

Atkins supports a GitHub Actions-inspired syntax, making it familiar for teams using GHA workflows.

## Syntax Comparison

**GitHub Actions:**
```yaml
name: Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: go build ./...
```

**Atkins:**
```yaml
name: Build

jobs:
  build:
    runs_on: ubuntu-latest
    steps:
      - name: Build
        run: go build ./...
```

## Key Differences

### No Triggers
Atkins doesn't handle CI triggers (`on:`). It's a command runner, not a CI system. Use it within your existing CI or as a local tool.

### No `uses:` Actions
Atkins doesn't support GitHub's action ecosystem. Replace `uses:` steps with equivalent commands:

**GHA:**
```yaml
- uses: actions/checkout@v4
- uses: actions/setup-go@v5
  with:
    go-version: '1.22'
```

**Atkins:**
```yaml
# Checkout is typically already done by CI
# Go setup depends on your environment
- run: go version
```

### Underscore vs Hyphen
Atkins uses underscores in field names for YAML compatibility:

| GitHub Actions | Atkins |
|---------------|--------|
| `runs-on` | `runs_on` |
| `depends-on` | `depends_on` |

## Jobs and Dependencies

**GitHub Actions:**
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: go test ./...

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - run: go build ./...
```

**Atkins:**
```yaml
jobs:
  test:
    steps:
      - run: go test ./...

  build:
    depends_on: test
    steps:
      - run: go build ./...
```

## Variables

**GitHub Actions:**
```yaml
env:
  MY_VAR: value

jobs:
  build:
    env:
      BUILD_VAR: ${{ env.MY_VAR }}
    steps:
      - run: echo $BUILD_VAR
```

**Atkins:**
```yaml
vars:
  my_var: value

jobs:
  build:
    env:
      vars:
        BUILD_VAR: ${{ my_var }}
    steps:
      - run: echo $BUILD_VAR
```

## Matrix Builds

GHA's matrix strategy maps to Atkins' `for:` loops:

**GitHub Actions:**
```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ['1.21', '1.22']
    runs-on: ${{ matrix.os }}
    steps:
      - run: go test ./...
```

**Atkins:**
```yaml
vars:
  go_versions:
    - '1.21'
    - '1.22'

jobs:
  test:
    steps:
      - for: version in go_versions
        task: test_version

  test_version:
    requires: [version]
    steps:
      - run: echo "Testing with Go ${{ version }}"
```

## Conditional Execution

**GitHub Actions:**
```yaml
- name: Deploy
  if: github.ref == 'refs/heads/main'
  run: ./deploy.sh
```

**Atkins:**
```yaml
- name: Deploy
  if: branch == "main"
  run: ./deploy.sh
```

Atkins uses [expr-lang](https://expr-lang.org/) for condition evaluation.

## Parallel Execution

**GitHub Actions** runs jobs in parallel by default.

**Atkins** runs jobs sequentially unless you use `detach: true`:

```yaml
jobs:
  lint:
    detach: true  # Run in background
    steps:
      - run: golangci-lint run

  test:
    detach: true  # Run in parallel with lint
    steps:
      - run: go test ./...

  build:
    depends_on: [lint, test]  # Wait for both
    steps:
      - run: go build ./...
```

## Best Practices

1. **Keep it simple**: Atkins is for running commands, not replacing CI
2. **Use for local dev**: Great for running the same tasks locally that CI runs
3. **Combine with CI**: Call `atkins` from your GHA workflow for consistency
