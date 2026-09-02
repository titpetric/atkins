# atkins - A pipeline runner with skills for CI/CD and development

Atkins is a minimal runner focused on usage in development, testing and
enables portability between CI/CD environments. It features a nice CLI
status tree, where you can see which jobs are running, and run jobs and
steps in parallel.

![](https://github.com/titpetric/atkins/blob/main/examples/atkins.list.png)

With atkins you can:

- define pipelines, jobs and steps and run them
- provide project or system skills via `.atkins/skills`
- run individual pipelines as executables
- attach to a CI/CD server with `atkins --login https://domain`, so every
  run you make locally is recorded as a job, and `atkins --dispatch` hands
  one to an agent instead

It's driven by yaml syntax, and supports shell interpolation with `$(...)`, and
yaml friendly variable interpolation: `name: ${{app.name}}`.

`atkins --help` prints the whole reference in one document: every command and
flag, the skills installed on the machine with the jobs each contributes, and
the atkins.yml schema. Redirected to a file it is markdown.

## Documentation

- Getting Started
  - [Introduction](https://atkins.incubator.to/getting-started/introduction)
  - [Installation](https://atkins.incubator.to/getting-started/installation)
  - [Why use Atkins?](https://atkins.incubator.to/getting-started/why-atkins)
- Reference
  - [Schema](https://atkins.incubator.to/reference/schema)
  - [Pipeline](https://atkins.incubator.to/reference/pipeline)
  - [Jobs](https://atkins.incubator.to/reference/jobs)
  - [Steps](https://atkins.incubator.to/reference/steps)
  - [Variables](https://atkins.incubator.to/reference/variables)
  - [Includes](https://atkins.incubator.to/reference/includes)
  - [Templating](https://atkins.incubator.to/reference/templating)
- Usage Guide
  - [Configuration](https://atkins.incubator.to/usage/configuration)
  - [Pipelines](https://atkins.incubator.to/usage/pipelines)
  - [Jobs](https://atkins.incubator.to/usage/jobs)
  - [Steps](https://atkins.incubator.to/usage/steps)
  - [Conditionals](https://atkins.incubator.to/usage/conditionals)
  - [Loops](https://atkins.incubator.to/usage/loops)
  - [Skills](https://atkins.incubator.to/usage/skills)
  - [CLI Flags](https://atkins.incubator.to/usage/cli-flags)
  - [Job Targeting](https://atkins.incubator.to/usage/job-targeting)
  - [Script Mode](https://atkins.incubator.to/usage/script-mode)
  - [Automation (JSON/YAML)](https://atkins.incubator.to/usage/automation)
  - [CI/CD Server](https://atkins.incubator.to/usage/ci-cd)
- Migrating
  - [Overview](https://atkins.incubator.to/migrating/migrating)
  - [From Taskfile](https://atkins.incubator.to/migrating/migration-from-task)
  - [From GitHub Actions](https://atkins.incubator.to/migrating/migration-from-github-actions)
