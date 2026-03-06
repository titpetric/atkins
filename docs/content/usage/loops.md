---
title: Loops
subtitle: For loops and iteration
layout: page
---

Use `for:` in steps to invoke tasks repeatedly with different loop variables.

## Basic Syntax

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

## How It Works

1. **`for:` in a step** - defines the loop with `for: variable in collection`
2. **`task:` in the same step** - names the task to invoke for each iteration
3. **Loop variable** - becomes available in the invoked task as `${{ variable }}`
4. **`requires:`** - the invoked task can declare required variables to validate they are present

## Example

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

## Missing Variables

If a required variable is missing, execution fails with a clear error:

```
job 'deploy_service' requires variables [env service_version] but missing: [env]
```

## See Also

- [Steps](./steps) - Step configuration
- [Jobs](./jobs) - Job configuration
