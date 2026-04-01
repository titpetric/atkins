# Atkins Agent

The agent is a full-screen interactive REPL for running tasks. Start it with `atkins --agent`.

For non-interactive use, run a single prompt with `-x`:

```
atkins -x "go:test"
atkins -x "hello"
atkins -x "uname -n"
```

![](structure.svg)

## Layout

The screen has three areas: a header bar (version, hostname), a scrollable message log, and a footer bar showing the working directory and git branch. Use `PgUp`/`PgDn` to scroll the log.

## Running Tasks

Type a task name to run it. Natural language is supported — filler words are stripped and keywords are matched against task names and descriptions.

```
go:test          → runs go:test directly
run tests        → matches "test" after stripping "run"
run go tests     → matches "go:test" by joining keywords
build            → runs build
```

Plural forms are handled automatically (`tests` → `test`, `queries` → `query`).

## Slash Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `/list` | `/l`, `/ls` | List available skills and jobs |
| `/run <task>` | `/r` | Run a specific task |
| `/cd <path>` | | Change working directory |
| `/help` | `/h`, `/?` | Show help |
| `/history` | | Show command history |
| `/clear` | `/cls` | Clear the message log |
| `/debug` | | Toggle debug mode |
| `/verbose` | `/v` | Toggle verbose output |
| `/jail` | | Toggle jail mode |
| `/quit` | `/q`, `/exit` | Exit |

## Shell Commands

If input doesn't match any task, the agent checks whether the first word is an executable on your system. If it is, the command runs as a shell command with output displayed in a blockquote-style border.

```
uname -n
ls -la
git status
```

Shell commands are recorded in `~/.atkins/shell_history.json` with the command, exit code, duration, and working directory. If you later type something that matches a single entry from shell history, it runs automatically.

## Greetings

The agent responds to greetings in several languages:

- **English**: hi, hey, hello, howdy, yo, sup
- **Spanish**: hola, buenas
- **French**: bonjour, salut
- **Italian**: ciao, salve
- **German**: hallo, moin, servus
- **Portuguese**: olá, oi

Responses come back in the matching language.

You can teach new greetings:

```
merhaba is a greeting
```

Custom greetings and responses are stored in `~/.atkins/greetings.yaml`.

## Fortune

Ask for a fortune in natural language:

```
fortune
give me a fortune
show me my fortune
inspire me
quote
```

Uses the system `fortune` command if available, otherwise returns a coding quote.

## Teaching Corrections

Teach the agent to map phrases to tasks:

```
if I say to run go test, run go:test
map lint to go:lint
alias build to go:build
go test means go:test
```

Corrections are stored in `~/.atkins/aliases.yaml` and checked before any other matching.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Submit input |
| `Up`/`Down` | Navigate command history |
| `PgUp`/`PgDn` | Scroll message log |
| `Ctrl+L` | Clear log |
| `Ctrl+A`/`Home` | Move cursor to start |
| `Ctrl+E`/`End` | Move cursor to end |
| `Ctrl+U` | Delete to start of line |
| `Ctrl+K` | Delete to end of line |
| `Ctrl+C` | Quit |

## Configuration Files

All stored under `~/.atkins/`:

| File | Purpose |
|------|---------|
| `aliases.yaml` | Phrase → task corrections |
| `greetings.yaml` | Custom greeting words and responses |
| `shell_history.json` | Shell command execution history |
