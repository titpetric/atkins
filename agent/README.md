# Atkins Agent

The agent is a full-screen interactive REPL for running tasks. Start it with `atkins --agent`.

The agent takes human language inputs, provides shell execution, navigation and pipeline job execution. Atkins skills can be used, and aliases can be created with the prompt. It requires no LLM.

For non-interactive use, run a single prompt with `-x`:

```
atkins -x "go:test"              # run a skill
atkins -x "curl -s wttr.in"      # run a shell command
atkins -x "list tasks"           # list available skills
atkins -x "hello"                # get a greeting
```

Aliases can be defined and invoked with language prompts. Teach an alias with `alias weather to curl -s wttr.in | head -n 7`, then invoke it with `how's the weather?`.

The routing flow is defined in `structure.d2`:

![](structure.svg)

## Layout

The screen has three areas: a header bar showing version and hostname, a scrollable message log, and a footer bar showing the working directory, git branch, and git stats. Use PgUp and PgDn to scroll the log.

The footer border is colorized in slate/teal and shows real-time git diff stats that update after each command. Stats show lines added and removed across both staged and unstaged changes.

### Prompt Modes

The prompt character indicates the current input mode:

| Prompt | Mode | Description |
|--------|------|-------------|
| `>` | Language | Natural language, task names, slash commands |
| `$` | Shell | Direct shell command execution |

Type `$` as the first character to switch to shell mode. The prompt displays in deep orange with command text in bright white. Backspace deletes the `$` to return to language mode.

## Running Tasks

Type a task name directly to run it. The agent also supports natural language where filler words like "run", "the", and "please" are stripped and remaining keywords are matched against task names and descriptions.

Typing `go:test` runs that task directly. Typing `run tests` strips "run" and matches "tests" to a test task. Typing `run go tests` joins keywords to match `go:test`. Plural forms are normalized automatically so `tests` matches `test` and `queries` matches `query`.

### Retry Failed Commands

After a command fails, re-run it by typing `again`, `retry`, or `redo`.

### Command Chaining

Run multiple tasks in sequence using `&&` or the word `then`. Typing `go:test && go:build` runs test first, then build if test passes. Typing `test then build` does the same using natural language.

### Typo Correction

If you mistype a command, the agent suggests corrections and asks for confirmation. Type `y` to accept the suggestion or `n` to cancel.

## Slash Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `/list` | | List available skills and jobs |
| `/run <task>` | | Run a specific task |
| `/aliases` | | List defined aliases |
| `/cd <path>` | | Change working directory |
| `/help` | `/h`, `/?` | Show help |
| `/history` | | Show command history |
| `/debug` | | Toggle debug mode |
| `/verbose` | `/v` | Toggle verbose output |
| `/jail` | | Toggle jail mode (restrict to project scope) |
| `/quit` | `/q`, `/exit` | Exit |

Slash commands can also be invoked using natural language. Typing `list`, `list tasks`, `tasks`, or `skills` all invoke `/list`. Typing `history` invokes `/history` and `help` invokes `/help`.

## Shell Commands

The agent supports two ways to run shell commands.

### Automatic Shell Detection

The agent detects shell commands by checking if the first word is an executable in PATH. Shell commands take precedence over natural language slash commands. Commands like `curl wttr.in`, `uname -n`, `ls -la`, `git status`, and `docker ps` are detected and executed directly.

### Explicit Shell Mode

Type `$` as the first character to enter shell mode. This changes the prompt from `>` to `$` displayed in deep orange. In shell mode, the command executes directly as a shell command without any routing logic. Use shell mode when a command name conflicts with a task name or when you want to ensure shell execution.

### Shell History

Shell commands are recorded in `~/.atkins/shell_history.json` with the command, exit code, duration, and working directory. If you later type something that matches a single entry from shell history, it runs automatically.

## Greetings

The agent responds to greetings and thanks in several languages including English, Spanish, French, Italian, German, and Portuguese. Responses come back in the matching language.

English greetings include hi, hey, hello, howdy, yo, and sup. Thanks include thanks, thank you, thx, and cheers. Other languages follow similar patterns with their native words.

Teach new greetings by typing something like `merhaba is a greeting`. Custom greetings are stored in `~/.atkins/greetings.yaml`.

## Fortune

Ask for a fortune by typing `fortune`, `give me a fortune`, `inspire me`, or `quote`. Uses the system `fortune` command if available, otherwise returns a coding quote.

## Teaching Aliases

Teach the agent to map phrases to tasks or shell commands using patterns like:

- `alias server name to uname -n`
- `alias weather to curl wttr.in`
- `if I say deploy, run docker:push`
- `map lint to go:lint`
- `deploy means docker:push`

Aliases work with natural language. After teaching `alias server name to uname -n`, typing `server name` or `what's your server name` both run `uname -n` because filler words are stripped.

Use `/aliases` to list all defined aliases. Aliases are stored in `~/.atkins/aliases.yaml` and checked before any other matching.

## Configuration Files

All configuration is stored under `~/.atkins/`:

| File | Purpose |
|------|---------|
| `aliases.yaml` | Phrase to task mappings |
| `greetings.yaml` | Custom greeting words and responses |
| `shell_history.json` | Shell command execution history |
