# Migrating from Taskfiles to Atkins

## Syntax differences

Taskfile and Atkins make some different choices as to the syntax of
shell invocations and the interpolation of template variables.

### Structure

Atkins supports a taskfile-like definition structure:

```yaml
vars:
  key: val

tasks:
  default:
    cmds:
      - cmd: ...
      - task: target
```

It also supports a similar GitHub Actions influenced syntax:

```yaml
jobs:
  default:
    steps:
      - run: ...
```

This allows some variance in style but not execution semantics.

### Shell invocations

When it comes to `task`, it invokes external commands via the `sh:`
field under a variable. This looks like so:

```yaml
vars:
  uname:
    sh: uname -n
```

The atkins variant uses the bash `$(expr)` as yaml valid syntax for the
execution of a command. The above example becomes:

```yaml
vars:
  uname: $(uname -n)
```

The expression can be used in interpolated statements, not just variables.

### Template variables

Taskfiles support Go templating to manage and interpolate variables.

The Task syntax for this causes various issues with YAML interpolation, and
requires variable references to be quoted as strings, e.g.:

```
vars:
  foo: bar
  bar: {{.foo}}
```

The value of `bar` needs to be quoted in task. Atkins chooses a yaml compatible
interpolation signature, and avoids the "." as well:

```yaml
vars:
  foo: bar
  bar: ${{foo}}
```

This doesn't require quoting the value.

Bash values like ${VAR} or $VAR are left to the shell.

### Environment

It's intended for atkins to run in a configured environment. You don't need
to declare or pass any environment variables specifically, and if you do define
some, you can define them on the pipeline level, job level or step level.

When using task, there are different design decisions made and task will
not inherit environment from execution, and make it a little bit difficult
to achieve a shared environment for all the executed tasks.

### Size

Atkins is smaller than Task with the most notable dependency being
`expr-lang`, used to evaluate `if` conditions and expressions.

Task has a few heavy dependencies like syntax highlighting that
contribute to the size of the package. I consider this a sign of
software bloat, because what good reason can you have to provide syntax
highlighting for an execution engine?

Atkins sticks to a minimal set of packages.

## Troubleshooting

As Atkins is opinionated on it's own syntax choices, Taskfile
configuration is not completely portable. For smaller taskfiles without
interpolation however, it's expected to work in the completely same
ways.

For example:

```yaml
tasks:
  up: docker compose up -d --remove-orphans
  down: docker compose down --remove-orphans
```

```text
$ task --list-all
task: Available tasks for this project:
* down:       
* up:         
```

```text
$ atkins -l -f Taskfile.yml
Taskfile.yml

* down:  docker compose down --remove-orphans
* up:    docker compose up -d --remove-orphans
```

Task has made some decisions that in my opinion impact usability, namely
to have task descriptions required in `-l` and not have environment
carry between task invocations. I have to use `--list-all` as shown
because `-l` doesn't work, and both of those outputs don't give any
detail.

Atkins aims to be low-impact in this case and gives a single way to list
tasks and surfaces detail about single cmd execution if no description
is given. A description is respected if filled.
