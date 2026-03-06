---
title: Conditionals
subtitle: Conditional execution with if expressions
layout: page
---

Jobs and steps can be conditionally executed using the `if:` field. Conditions are evaluated using [expr-lang](https://expr-lang.org/), a simple expression language.

## Basic Syntax

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
| `                    |             | `                            |
| `!`                  | Logical NOT | `!skip_tests`                |
| `>`, `<`, `>=`, `<=` | Comparisons | `num_retries > 0`            |
| `in`                 | Contains    | `"prod" in environments`     |
| `matches`            | Regex match | `branch matches "^release/"` |

## Examples

**Combining conditions:**

```yaml
if: environment == "production" && branch == "main"
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
```

## Truthiness

Values are coerced to boolean as follows:

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

Undefined variables evaluate to `nil` (falsy) rather than causing an error:

```yaml
jobs:
  optional:
    if: maybe_defined
    steps:
      - run: ./optional.sh
```

## Skipped Output

When a job or step is skipped due to a condition, the tree output shows the condition:

```
[ok] build
[skip] deploy (if: environment == "production")
```

## See Also

- [Jobs](./jobs) - Job configuration
- [Steps](./steps) - Step configuration
