---
title: Job Targeting
subtitle: Running specific jobs with targeting syntax
layout: page
---

# Job Targeting

Atkins provides flexible ways to target and run specific jobs, including cross-pipeline references.

## Basic Targeting

### Run by Name

```bash
# Run a job by its name
atkins build

# Equivalent to
atkins --job build
```

### Namespaced Jobs

Jobs from skills use `skill:job` syntax:

```bash
# Run 'test' job from 'go' skill
atkins go:test

# Run 'build' job from 'docker' skill
atkins docker:build
```

## The Default Job

When running `atkins` without arguments:

1. Looks for a job named `default`
2. Falls back to a job with `default` in its aliases

```yaml
jobs:
  all:
    aliases: [default]
    depends_on: [lint, test, build]
```

If no default is found, Atkins shows available jobs:

```
atkins: Available jobs for this project:
* build:       Build the application
* test:        Run tests
atkins: Job "default" does not exist
```

## Root Job Targeting (`:` Prefix)

The `:` prefix bypasses alias resolution and targets jobs directly.

### Target Main Pipeline

```bash
# Target 'build' in main pipeline (bypasses aliases)
atkins :build
```

This is useful when:
- A skill has aliased `build` but you want the main pipeline's `build`
- You want explicit, unambiguous job targeting

### Target Skill Pipeline

```bash
# Target 'build' job in 'go' skill explicitly
atkins :go:build

# Target 'test' job in 'docker' skill
atkins :docker:test
```

## Cross-Pipeline Task References

Within a pipeline, steps can reference tasks from other pipelines using `:` syntax.

### Reference Main Pipeline

```yaml
# In a skill pipeline
jobs:
  deploy:
    steps:
      # Call 'build' from main pipeline
      - task: :build
      - run: ./deploy.sh
```

### Reference Other Skills

```yaml
# In release skill
jobs:
  release:
    steps:
      # Call tasks from other skills
      - task: :go:build
      - task: :docker:build
      - task: :docker:push
```

### Example: Multi-Skill Coordination

**Main pipeline (atkins.yml):**
```yaml
name: My App

jobs:
  build:
    steps:
      - run: go build -o app

  deploy:
    steps:
      - task: :go:test      # From go skill
      - task: :docker:build  # From docker skill
      - run: kubectl apply -f k8s/
```

**Go skill (.atkins/skills/go.yml):**
```yaml
name: Go Skill
when:
  files: [go.mod]

jobs:
  test:
    steps:
      - run: go test ./...
```

**Docker skill (.atkins/skills/docker.yml):**
```yaml
name: Docker Skill
when:
  files: [Dockerfile]

jobs:
  build:
    steps:
      - run: docker build -t app .
```

## Aliases

Jobs can have alternative names:

```yaml
jobs:
  docker:build:
    aliases: [build, b, db]
    steps:
      - run: docker build -t app .
```

Now all of these work:

```bash
atkins docker:build  # Full name
atkins build         # Alias
atkins b             # Short alias
atkins db            # Another alias
```

### Job Resolution Order

When you invoke `atkins <name>`, the resolution follows this precedence:

1. **Explicit root reference** (`:` prefix) - bypasses all other rules
2. **Prefixed job reference** (`skill:job` syntax) - explicit skill targeting
3. **Exact main pipeline match** - job name matches exactly in main pipeline
4. **Alias match** - job alias in any pipeline
5. **Skill ID with default** - name matches skill with `default` job
6. **Skill ID** (listing only) - name matches skill name
7. **Fuzzy match** - substring/suffix match (single match only)
8. **Fallback** - main pipeline with name as-is

**Key behavior:** Main pipeline jobs take precedence over aliases. If your main pipeline has a job named `up`, running `atkins up` will invoke it even if a skill has an alias `up` pointing elsewhere.

## Fuzzy Matching

If no exact match is found, Atkins tries fuzzy matching:

```bash
# If 'docker:build' exists
atkins build  # Matches via suffix
```

When multiple matches exist:

```
INFO: found 2 matching jobs:

  - go:build
  - docker:build
```

Use the full name or `:` prefix to disambiguate.

## Nested Jobs

Jobs with `:` in their name are nested and not directly executable:

```yaml
jobs:
  build:
    steps:
      - task: build:linux
      - task: build:darwin

  build:linux:    # Nested - only runs via 'build'
    steps:
      - run: GOOS=linux go build

  build:darwin:   # Nested - only runs via 'build'
    steps:
      - run: GOOS=darwin go build
```

```bash
atkins build         # Runs build:linux and build:darwin
atkins build:linux   # Error - nested job
```

## Targeting Examples

### Common Invocations

```bash
# Run default job
atkins

# Run specific job from main pipeline
atkins test
atkins up

# Run skill job (explicit)
atkins go:lint
atkins docker:build
```

### Root Reference (`:` prefix)

```bash
# Force main pipeline job (bypasses aliases)
atkins :up
atkins :build

# Force skill job (explicit targeting)
atkins :go:build
atkins :docker:push
```

### When to Use `:` Prefix

Use the `:` prefix when:
- You want to ensure the main pipeline job runs (not an alias)
- You need explicit, unambiguous targeting
- A skill has aliased a common name you want to bypass

```bash
# Main pipeline has 'up' job, skill has 'up' alias → runs main pipeline
atkins up

# Force main pipeline 'up' even if you're unsure about aliases
atkins :up
```

### Full Example

```bash
# Run default job
atkins

# Run 'build' from main pipeline
atkins build

# Run 'test' from go skill
atkins go:test

# Run 'up' from main pipeline (explicit)
atkins :up

# Run 'build' from docker skill (explicit)
atkins :docker:build

# Using alias (if 'b' is alias for build)
atkins b

# With file flag
atkins -f ci/test.yml integration
```
