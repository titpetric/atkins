---
title: Skills
subtitle: Modular, reusable pipeline components
layout: page
---

# Skills

Skills are modular pipeline components that automatically activate based on project context. They enable reusable workflows across projects.

## What Are Skills?

Skills are YAML pipeline files stored in special directories:

- `.atkins/skills/` - Project-local skills
- `$HOME/.atkins/skills/` - Global skills (shared across projects)

Each skill file becomes a namespace. For example, `go.yml` creates jobs like `go:build`, `go:test`.

## Skill Locations

### Project Skills

```
myproject/
├── .atkins/
│   └── skills/
│       ├── go.yml        → go:*
│       ├── docker.yml    → docker:*
│       └── deploy.yml    → deploy:*
└── atkins.yml
```

### Global Skills

```
$HOME/
└── .atkins/
    └── skills/
        ├── go.yml        → Available in all projects
        └── node.yml      → Available in all projects
```

Project skills take precedence over global skills with the same name.

## Creating a Skill

**`.atkins/skills/go.yml`:**
```yaml
name: Go build and test

when:
  files:
    - go.mod

jobs:
  default:
    desc: Go lifecycle
    depends_on: [fmt, lint, test, build]

  fmt:
    desc: Format code
    steps:
      - run: gofmt -w .

  lint:
    desc: Run linter
    steps:
      - run: golangci-lint run

  test:
    desc: Run tests
    steps:
      - run: go test ./...

  build:
    desc: Build binary
    aliases: [build]  # Creates global 'build' alias
    steps:
      - run: go build -o bin/app ./...
```

## Conditional Activation

The `when` block controls when a skill is available.

### File-Based Conditions

```yaml
when:
  files:
    - go.mod          # Activate if go.mod exists
```

```yaml
when:
  files:
    - package.json    # Activate for Node projects
    - yarn.lock
```

Multiple files use OR logic - any match activates the skill.

### Dynamic File Patterns

```yaml
when:
  files:
    - $(find . -name "*.go" | head -1)  # Shell command
```

### Path Searching

File patterns search upward from the current directory:

```yaml
when:
  files:
    - Dockerfile      # Finds Dockerfile in cwd or parent dirs
```

## Skill Namespacing

Skills automatically namespace their jobs:

**go.yml** creates:
- `go:default`
- `go:build`
- `go:test`

Access them with:
```bash
atkins go:build
atkins go:test
```

## Aliases

Skills can provide global aliases:

```yaml
jobs:
  build:
    aliases: [build, b]
    steps:
      - run: go build
```

Now `atkins build` invokes `go:build`.

### Alias Conflicts

If multiple skills define the same alias, the first-loaded wins (project before global).

To target explicitly:
```bash
atkins :go:build      # Explicit skill reference
atkins :docker:build  # Different skill
```

## Default Jobs

A skill can have a `default` job, enabling shorthand invocation:

```yaml
jobs:
  default:
    depends_on: [lint, test, build]
```

```bash
atkins go        # Runs go:default
```

## Cross-Skill References

Skills can reference each other using `:skill:job` syntax:

**release.yml:**
```yaml
jobs:
  release:
    steps:
      - task: :go:test
      - task: :go:build
      - task: :docker:build
      - task: :docker:push
```

## Skill Variables

Skills have their own variable scope:

```yaml
name: Docker Skill

vars:
  image: myapp
  registry: docker.io

jobs:
  build:
    steps:
      - run: docker build -t ${{ registry }}/${{ image }} .
```

## Example Skills

### Go Skill

```yaml
name: Go build and test
when:
  files: [go.mod]

vars:
  binary: $(basename $(pwd))

jobs:
  default:
    depends_on: [generate, fmt, lint, test, build]

  generate:
    steps:
      - run: go generate ./...

  fmt:
    steps:
      - run: gofmt -w .
      - run: goimports -w .

  lint:
    steps:
      - run: golangci-lint run

  test:
    aliases: [test]
    steps:
      - run: go test ./...

  build:
    aliases: [build]
    steps:
      - run: go build -o bin/${{ binary }} ./...
```

### Docker Skill

```yaml
name: Docker build and push
when:
  files: [Dockerfile]

vars:
  image: $(basename $(pwd))
  tag: $(git describe --tags --always)

jobs:
  build:
    aliases: [docker]
    steps:
      - run: docker build -t ${{ image }}:${{ tag }} .

  push:
    steps:
      - run: docker push ${{ image }}:${{ tag }}
```

### Node.js Skill

```yaml
name: Node.js
when:
  files: [package.json]

jobs:
  install:
    steps:
      - run: npm install

  build:
    depends_on: [install]
    steps:
      - run: npm run build

  test:
    depends_on: [install]
    aliases: [test]
    steps:
      - run: npm test
```

## Jail Mode

To disable global skills:

```bash
atkins --jail
```

This only loads skills from `.atkins/skills/`, ignoring `$HOME/.atkins/skills/`.

Useful for:
- Reproducible CI builds
- Avoiding personal customizations
- Testing project-only configurations

## Listing Skills

View all active skills and their jobs:

```bash
atkins -l
```

Output shows skills after the main pipeline:

```
My Project

* default:    Run all
* build:      Build app

Aliases

* go:         (invokes: go:default)
* test:       (invokes: go:test)

Go build and test

* go:default: Go lifecycle
* go:build:   Build binary
* go:test:    Run tests
```
