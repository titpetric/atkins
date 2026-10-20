---
title: Conditionals
subtitle: Conditional execution with if expressions
layout: page
---

Jobs and steps can be conditionally executed using the `if:` field. Atkins evaluates conditions with [expr-lang](https://expr-lang.org/), a simple expression language. The `if:` field accepts a single expression or a list of expressions. For a list, all conditions must be true (AND logic).

## Examples

@tabs
@file "Job Conditionals" conditionals/job-if.yml
@file "Step Conditionals" conditionals/step-if.yml

![Job Conditionals](./conditionals/job-if.png)

## Available Variables

Conditions have access to:

1. **Pipeline variables** - All variables defined in `vars:` blocks
2. **Environment variables** - All environment variables (from shell and `env:` blocks)
3. **Loop variables** - When inside a `for:` loop, the loop variable is available

## Expression Syntax

Expr-lang supports common operators and comparisons:

| Operator             | Description | Example                      |
|----------------------|-------------|------------------------------|
| `==`                 | Equals      | `env == "prod"`              |
| `!=`                 | Not equals  | `env != "dev"`               |
| `&&`                 | Logical AND | `a == 1 && b == 2`           |
| `&#124;&#124;`       | Logical OR  | `a == 1 &#124;&#124; b == 2` |
| `!`                  | Logical NOT | `!skip_tests`                |
| `>`, `<`, `>=`, `<=` | Comparisons | `num_retries > 0`            |
| `in`                 | Contains    | `"prod" in environments`     |
| `matches`            | Regex match | `branch matches "^release/"` |

## Examples

**Combining conditions (inline):**

```yaml
if: environment == "production" && branch == "main"
```

**Combining conditions (list form):**

```yaml
if:
  - branch == "main"
  - environment == "production"
```

Both forms behave the same. The list form helps when conditions are long or many.

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
      - run: echo "Deploying..."
```

**Pattern matching:**

```yaml
if: branch matches "^release/.*"
```

**Checking for files and directories:**

```yaml
jobs:
  build-docs:
    if: $(test -f README.md)
    steps:
      - run: echo "README.md exists"

  publish-docs:
    if: $(test -d docs)
    steps:
      - run: echo "docs directory exists"
```

Use `test -f` to check for a regular file and `test -d` to check for a
directory. Commands in `$(...)` use their exit status as a condition when they
produce no output: exit status 0 evaluates to `true`, and a non-zero exit status
evaluates to `false` without failing the job. A non-zero status always evaluates
to `false`, even if the command wrote output.

## Truthiness

Atkins coerces values to boolean as follows:

| Value               | Result |
|---------------------|--------|
| `true`              | true   |
| `false`             | false  |
| `nil` / undefined   | false  |
| `""` (empty string) | false  |
| `"false"`, `"0"`    | false  |
| Any other string    | true   |
| `0`                 | false  |
| Any other number    | true   |

## Undefined Variables

Undefined variables resolve to `nil` (falsy) and cause no error:

```yaml
jobs:
  optional:
    if: maybe_defined
    steps:
      - run: echo "Running optional job..."
```

## Jobs with `for:` and `if:`

When a job has both a `for:` loop and an `if:` condition, Atkins checks the `if:` expression per iteration. This allows filtering iterations based on loop variables:

```yaml
jobs:
  deploy:
    for: env in environments
    if: env != "dev"
    steps:
      - run: echo "Deploying to ${{ env }}..."
```

In this example, Atkins skips iterations where `env` is `"dev"` and runs all other environments.

## Skipped Output

When a condition skips a job or step, the tree output shows the condition:

```text
[ok] build
[skip] deploy (if: environment == "production")
```

## See Also

- [Jobs](./jobs) - Job configuration
- [Steps](./steps) - Step configuration
