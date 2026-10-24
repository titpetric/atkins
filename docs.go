// Command atkins runs the jobs of a pipeline file and records each run.
//
// atkins finds `.atkins.yml`, a Taskfile, or the file of `-f`, from the
// working directory up, loads the skills that apply there and
// runs the jobs of the arguments, or `default`, as a tree it redraws
// on a terminal. `--lint` checks a pipeline, `-l` lists what it declares,
// `--help` prints the flags, the skills and the schema in one document.
//
// Every run records its commands into two sqlite databases, the project's
// and the workspace's under `$HOME/.atkins`, or the file `ATKINS_DB` names
// with the mirrors listed after it; `--local` records nowhere.
// `--stats` reads them back and reports where the time goes: which command
// costs the most once its frequency counts, by day, by tool and by command
// key, as tables on a terminal, as markdown in a pipe, or as JSON and YAML
// for a program.
//
// `--vendor` lists the skills from `$HOME/.atkins/skills` the repository
// uses, and `--vendor --write` copies them into `.atkins/skills`.
// A menu edits `.atkins/config.yml`, and the help document lists its flag.
package main
