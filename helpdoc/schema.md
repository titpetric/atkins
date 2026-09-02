A pipeline file declares jobs; a job runs steps. `atkins --lint` checks a file, `atkins -l` lists what it declares.

### One idea, several spellings

atkins reads the GitHub Actions vocabulary and the Taskfile vocabulary as one schema, so a pipeline states intent rather than picking a dialect. Each pair below is the same field. Both spellings may appear in the same file; set on the same object, the first column wins.

| Key              | Same as   | Where     | Note                                     |
|------------------|-----------|-----------|------------------------------------------|
| `jobs`           | `tasks`   | pipeline  | map of job name to job                   |
| `steps`          | `cmds`    | job       | list of steps                            |
| `run`            | `cmd`     | job, step | one shell command                        |
| `deferred: true` | `defer:`  | step      | `defer: X` is `{run: X, deferred: true}` |
| `x in xs`        | `xs as x` | `for`     | normalised to `x in xs`                  |

`cmds` is the one key whose meaning depends on where it sits: a list of steps on a job, a list of shell commands on a step.

Four more shorthands, in the same spirit:

- A job may be a bare string. `up: docker compose up` is a job of one step.
- A step may be a bare string. `- go build ./...` is `{run: go build ./...}`.
- `if`, `for`, `depends_on` and `include` each take one string or a list of strings. A list of `if` conditions is ANDed.
- A `for` iterator may bind the index: `(i, item) in items`.

### Top level

| Key             | Type         | Meaning                                                 |
|-----------------|--------------|---------------------------------------------------------|
| `name`          | string       | pipeline name, shown in `-l`; defaults to the file name |
| `help`          | string       | what the file is for, shown in `--help`; used by skills |
| `dir`           | string       | working directory for every job                         |
| `jobs`, `tasks` | map          | the jobs, keyed by the name `atkins <name>` runs        |
| `vars`          | map          | variables, read as `${{ name }}`                        |
| `env`           | object       | `{vars, include}`, exported into every command          |
| `include`       | string, list | pipeline files merged into this one                     |
| `when`          | object       | `{files: [...]}`, the paths that activate a skill       |

### A job

| Key             | Type         | Default        | Meaning                                             |
|-----------------|--------------|----------------|-----------------------------------------------------|
| `desc`          | string       | none           | one line shown by `-l`                              |
| `steps`, `cmds` | list         | none           | the steps to run, in order                          |
| `run`, `cmd`    | string       | none           | one command; becomes the job's only step            |
| `depends_on`    | string, list | none           | jobs that run before this one                       |
| `if`            | string, list | none           | run only when every condition holds                 |
| `for`           | string, list | none           | run the job once per item                           |
| `requires`      | list         | none           | variables that must be set when invoked in a loop   |
| `aliases`       | list         | none           | other names `atkins <name>` accepts                 |
| `vars`          | map          | none           | variables for this job and its steps                |
| `env`           | object       | none           | environment for this job and its steps              |
| `include`       | string, list | none           | files merged into this job                          |
| `dir`           | string       | pipeline `dir` | working directory                                   |
| `timeout`       | string       | none           | duration, e.g. `10m`, `300s`                        |
| `detach`        | bool         | false          | start in the background, join before deferred steps |
| `show`          | bool         | root jobs      | show the job in the tree                            |
| `passthru`      | bool         | false          | print command output under the tree                 |
| `tty`           | bool         | false          | allocate a PTY for every step, which keeps colour   |
| `interactive`   | bool         | false          | stream output live and connect stdin                |
| `summarize`     | bool         | false          | collapse output to the last lines                   |
| `quiet`         | bool         | false          | drop the output                                     |

### A step

| Key                 | Type         | Default     | Meaning                              |
|---------------------|--------------|-------------|--------------------------------------|
| `name`              | string       | the command | label in the tree                    |
| `desc`              | string       | none        | label in the tree, over `name`       |
| `run`, `cmd`        | string       | none        | one shell command                    |
| `cmds`              | list         | none        | shell commands run in sequence       |
| `task`              | string       | none        | jobs to invoke, whitespace separated |
| `if`                | string, list | none        | run only when every condition holds  |
| `for`               | string, list | none        | run the step once per item           |
| `vars`              | map          | none        | variables for this step              |
| `env`               | object       | none        | environment for this step            |
| `include`           | string, list | none        | files merged into this step          |
| `dir`               | string       | job `dir`   | working directory                    |
| `deferred`, `defer` | bool, step   | false       | run at the end of the job            |
| `detach`            | bool         | false       | start in the background              |
| `verbose`           | bool         | false       | print the output                     |
| `passthru`          | bool         | false       | print the output under the tree      |
| `tty`               | bool         | false       | allocate a PTY, which keeps colour   |
| `interactive`       | bool         | false       | stream output live and connect stdin |
| `summarize`         | bool         | false       | collapse output to the last lines    |
| `quiet`             | bool         | false       | drop the output                      |

A step names a command or a job, not both: `run`, `cmd`, `cmds` and `task` are alternatives, read in the order `task`, `run`, `cmd`, `cmds`.

A deferred step is declared where it makes sense to read it, usually first, and runs last. Every plain step of the job runs, then the detached ones are joined, then the deferred ones run in declaration order. A job that fails still runs them.

```yaml
jobs:
  migrate:
    steps:
      - defer: rm -f app.db
      - mig migrate --apply
```

### Values

- `${{ name }}` reads a variable. The value may come from `vars` at any level, from a loop, or from the caller when a job is invoked with `task:`.
- `$(command)` runs a shell command and substitutes its output. It is evaluated where it is written, including inside a `vars` value and inside an `if`.
- `${NAME}` is an environment variable, expanded by the shell that runs the command.
- An `if` condition is an expression over variables (`enabled == true`) or a shell command whose exit status decides (`$(test -f go.mod)`).
