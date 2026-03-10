# Atkins - portable command runner / CI tooling

Atkins is a minimal runner focused on usage in local testing and
CI/CD environments. It features a nice CLI status tree, where you can
see which jobs are running, and run jobs and steps in parallel.

The log of Atkins executions can be used for further processing, like rendering UML flow charts.
A few examples are given in the [./scripts folder](./scripts).

## Documentation

- Getting Started
  - [Introduction](./docs/content/getting-started/introduction.md) — Overview and features
  - [Installation](./docs/content/getting-started/installation.md) — How to install
  - [Migrating to Atkins](./docs/content/getting-started/migrating.md) — Why and how to migrate
    - [Migration from Task](./docs/content/getting-started/migration-from-task.md)
    - [Migration from GitHub Actions](./docs/content/getting-started/migration-from-github-actions.md)
- Usage Examples
  - [Configuration](./docs/content/usage/configuration.md) — Pipeline format and syntax
  - [Pipelines, Jobs and Steps](./docs/content/usage/pipelines-jobs-steps.md) — Execution hierarchy
  - [Skills](./docs/content/usage/skills.md) — Modular, reusable pipeline components
  - [CLI Flags](./docs/content/usage/cli-flags.md) — Command-line reference
  - [Job Targeting](./docs/content/usage/job-targeting.md) — Running specific jobs
  - [Running in Script Mode](./docs/content/usage/script-mode.md) — Executable pipelines and stdin
  - [Automation (JSON/YAML)](./docs/content/usage/automation.md) — Machine-readable output
- [Why use Atkins?](./docs/content/why-atkins.md) — Comparison with GHA, Taskfile, Lefthook

See pipeline examples in [./tests](./tests).

## Task Invocation with For Loops

You can invoke tasks within a for loop to run the same task multiple times with different loop variables. This is useful for processing multiple items in parallel.

### Basic Example

```yaml
vars:
  components:
    - src/main
    - src/utils
    - tests/

tasks:
  build:
    desc: "Build all components"
    steps:
      - for: component in components
        task: build_component

  build_component:
    desc: "Build a single component"
    requires: [component]  # Declare required variables
    steps:
      - run: make build -C "${{ component }}"
```

### How It Works

1. **For Loop in Step**: A step can have both `for:` and `task:` fields
   - `for: variable in collection` - Defines the loop pattern
   - `task: task_name` - Names the task to invoke

2. **Loop Variables**: The loop variable becomes available to the invoked task
   - Use `${{ variable_name }}` to reference the loop variable
   - Loop variables are merged with existing context variables

3. **Requires Declaration**: Tasks can declare required variables with `requires: [...]`
   - When invoked in a loop, the task validates that all required variables are present
   - If a required variable is missing, execution fails with a clear error message
   - `requires` is optional; omit it if the task doesn't require specific variables

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
    desc: "Deploy to all environments"
    steps:
      - for: env in environments
        task: deploy_service

  deploy_service:
    desc: "Deploy to a specific environment"
    requires: [env, service_version]
    steps:
      - run: kubectl apply -f config/${{ env }}/deployment.yml --image=app:${{ service_version }}
```

### Error Handling

If a task has `requires: [var1, var2]` but one of those variables is missing from the loop context:

```text
job 'deploy_service' requires variables [env service_version] but missing: [env]
```

The execution stops with a clear message listing the missing variables.
