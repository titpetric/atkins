# Atkins Agent

The agent is a full-screen interactive REPL for running tasks. Start it with `atkins --agent`.

For non-interactive use, run a single prompt with `-x`:

```
atkins -x "go:test"              # run a skill
atkins -x "curl -s wttr.in"      # run a shell command
atkins -x "list tasks"           # list available skills
atkins -x "hello"                # get a greeting
```

Aliases can be defined with language:

- Teach an alias: `alias weather to curl -s wttr.in | head -n 7`
- Run an alias: `how's the weather?'

The routing flow is defined in `structure.d2`:

![](structure.svg)

## Layout

The screen has three areas: a header bar (version, hostname), a scrollable message log, and a footer bar showing the working directory, git branch, and git stats (+/- line changes). Use `PgUp`/`PgDn` to scroll the log.

The footer border is colorized (slate/teal) and shows real-time git diff stats that update after each command.

## Running Tasks

Type a task name to run it. Natural language is supported — filler words are stripped and keywords are matched against task names and descriptions.

```
go:test          → runs go:test directly
run tests        → matches "test" after stripping "run"
run go tests     → matches "go:test" by joining keywords
build            → runs build
```

Plural forms are handled automatically (`tests` → `test`, `queries` → `query`).

### Retry Failed Commands

After a command fails, you can re-run it with:

```
again                → re-run the last command
retry                → same as again
redo                 → same as again
```

### Command Chaining

Run multiple tasks in sequence:

```
go:test && go:build          → run test, then build if test passes
test then build              → natural language chaining
build && test && deploy      → chain multiple tasks
```

### Typo Correction

If you mistype a command, the agent suggests corrections:

```
> tets
Did you mean go:test? [y/n]
```

Type `y` to confirm or `n` to cancel.

## Slash Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `/list` | `/l`, `/ls` | List available skills and jobs |
| `/run <task>` | `/r` | Run a specific task |
| `/aliases` | `/alias` | List defined aliases |
| `/cd <path>` | | Change working directory |
| `/help` | `/h`, `/?` | Show help |
| `/history` | | Show command history |
| `/clear` | `/cls` | Clear the message log |
| `/debug` | | Toggle debug mode |
| `/verbose` | `/v` | Toggle verbose output |
| `/jail` | | Toggle jail mode |
| `/quit` | `/q`, `/exit` | Exit |

Slash commands can also be invoked using natural language:

```
list              → /list
list tasks        → /list
tasks             → /list
skills            → /list
history           → /history
help              → /help
```

## Shell Commands

The agent detects shell commands by checking if the first word is an executable in your PATH. Shell commands take precedence over natural language slash commands.

```
curl wttr.in
uname -n
ls -la
git status
docker ps
```

Shell commands are recorded in `~/.atkins/shell_history.json` with the command, exit code, duration, and working directory. If you later type something that matches a single entry from shell history, it runs automatically.

## Greetings

The agent responds to greetings and thanks in several languages:

**Greetings:**
- **English**: hi, hey, hello, howdy, yo, sup
- **Spanish**: hola, buenas
- **French**: bonjour, salut
- **Italian**: ciao, salve
- **German**: hallo, moin, servus
- **Portuguese**: olá, oi

**Thanks:**
- **English**: thanks, thank you, thx, cheers
- **Spanish**: gracias, muchas gracias
- **French**: merci, merci beaucoup
- **Italian**: grazie, grazie mille
- **German**: danke, danke schön, vielen dank
- **Portuguese**: obrigado, obrigada, valeu

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

## Teaching Aliases

Teach the agent to map phrases to tasks or shell commands:

```
alias server name to uname -n
alias weather to curl wttr.in
if I say deploy, run docker:push
map lint to go:lint
alias build to go:build
go test means go:test
```

Aliases work with natural language — filler words are stripped:

```
server name           → runs uname -n
what's your server name → stripped to "server name" → runs uname -n
```

Use `/aliases` to list all defined aliases. Aliases are stored in `~/.atkins/aliases.yaml` and checked before any other matching.

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
