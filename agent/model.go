package agent

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/agent/router"
	"github.com/titpetric/atkins/agent/view"
	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
)

// UsageText returns the usage help text for non-interactive mode.
func UsageText() string {
	var b strings.Builder
	b.WriteString("atkins - task runner and shell assistant\n\n")
	b.WriteString("Usage:\n")
	b.WriteString("  atkins              Start interactive REPL\n")
	b.WriteString("  atkins -x \"<cmd>\"   Execute a single command\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("  atkins -x \"go:test\"             Run a skill\n")
	b.WriteString("  atkins -x \"curl wttr.in\"        Run shell command\n")
	b.WriteString("  atkins -x \"run the tests\"       Natural language\n")
	b.WriteString("  atkins -x \"list\"                List available skills\n\n")
	b.WriteString("Teach aliases:\n")
	b.WriteString("  alias server name to uname -n\n")
	b.WriteString("  if i say deploy, run docker:push\n")
	return b.String()
}

// Type aliases for view package types.
type (
	LogEntry   = view.LogEntry
	Breadcrumb = view.Breadcrumb
	PromptMode = view.PromptMode
	JobStatus  = view.JobStatus
	JobEntry   = view.JobEntry
	StepEntry  = view.StepEntry
	JobView    = view.JobView
)

// PromptMode constants.
const (
	PromptModeLanguage = view.PromptModeLanguage
	PromptModeShell    = view.PromptModeShell
)

// JobStatus constants.
const (
	JobStatusPending = view.JobStatusPending
	JobStatusRunning = view.JobStatusRunning
	JobStatusPassed  = view.JobStatusPassed
	JobStatusFailed  = view.JobStatusFailed
	JobStatusSkipped = view.JobStatusSkipped
)

// NewBreadcrumb creates a new breadcrumb tracker.
func NewBreadcrumb() *Breadcrumb {
	return view.NewBreadcrumb()
}

// DetectPromptMode returns the appropriate mode based on input.
func DetectPromptMode(input string) PromptMode {
	return view.DetectPromptMode(input)
}

// NewJobView creates a new job view.
func NewJobView() *JobView {
	return view.NewJobView()
}

// Model is the bubbletea model for the agent REPL.
type Model struct {
	agent      *Agent
	state      State
	input      string
	cursor     int
	history    []string
	historyIdx int
	breadcrumb *Breadcrumb
	lastError  error
	lastTask   *model.ResolvedTask
	retryCount int

	// Centralized router and slash commands
	router   *router.Router
	registry *Registry

	// Dimensions
	width  int
	height int

	// TUI state
	version   string
	hostname  string
	cwd       string
	gitBranch string
	gitStats  GitStats

	log       []LogEntry
	scrollOff int
	spinner   spinner.Model
	runLogIdx int // index of the current running entry in log

	// Confirmation state for fuzzy matching
	pendingConfirm *router.Route

	// Prompt mode (language or shell)
	promptMode PromptMode
}

// NewModel creates a new bubbletea model for the agent.
func NewModel(agent *Agent, version string) Model {
	cwd := agent.WorkDir()
	s := spinner.New()
	s.Spinner = spinner.Dot
	registry := DefaultRegistry()
	m := Model{
		agent:      agent,
		state:      StateIdle,
		history:    []string{},
		historyIdx: -1,
		breadcrumb: NewBreadcrumb(),
		router:     router.NewRouter(agent.Resolver(), agent.Pipelines(), registry),
		registry:   registry,
		version:    version,
		hostname:   detectHostname(),
		cwd:        cwd,
		gitBranch:  detectGitBranch(cwd),
		gitStats:   detectGitStats(cwd),
		spinner:    s,
		runLogIdx:  -1,
	}
	m.appendGreeting()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Model) appendGreeting() {
	m.appendLog("info", "")
	m.appendLog("info", colors.BrightCyan("Welcome to atkins.")+" Type a command to get started.")
	m.appendLog("info", "")
	m.appendLog("info", colors.Dim("Usage:"))
	m.appendLog("info", "  "+colors.BrightWhite("Natural language:")+
		"   \"run the tests\", \"build it\", \"list tasks\"")
	m.appendLog("info", "  "+colors.BrightWhite("Direct skills:")+
		"       go:test, build, test")
	m.appendLog("info", "  "+colors.BrightWhite("Shell commands:")+
		"     curl wttr.in, ls -la, docker ps")
	m.appendLog("info", "")
	m.appendLog("info", colors.Dim("Aliasing commands and job targets:"))
	m.appendLog("info", "  "+colors.BrightWhite("\"alias server name to uname -n\""))
	m.appendLog("info", "  "+colors.BrightWhite("\"if i say deploy, run docker:push\""))
	m.appendLog("info", "")
	m.appendLog("info", colors.Dim("Slash commands:")+
		"  /help  /list  /run <task>  /cd <path>  /quit")

	m.appendLog("info", "")
}

func (m *Model) appendLog(kind, text string) {
	m.log = append(m.log, LogEntry{
		Kind: kind,
		Text: text,
	})
	if kind == "prompt" {
		m.log = append(m.log, LogEntry{Kind: "info"})
	}
	m.scrollOff = 0 // follow tail
}

func (m *Model) appendRunLog(task string) int {
	m.log = append(m.log, LogEntry{
		Kind:    "run",
		Task:    task,
		Running: true,
	})
	m.scrollOff = 0
	return len(m.log) - 1
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ExecutionStartMsg:
		m.state = StateExecuting
		m.lastTask = msg.Resolved
		m.lastError = nil
		m.retryCount = 0
		m.breadcrumb.Clear()
		m.breadcrumb.Push(msg.Task)
		m.breadcrumb.SetStatus("running...")

		m.runLogIdx = m.appendRunLog(msg.Task)

		return m, tea.Batch(m.spinner.Tick, m.runPipeline(msg.Resolved))

	case ExecutionDoneMsg:
		if m.runLogIdx >= 0 && m.runLogIdx < len(m.log) {
			entry := &m.log[m.runLogIdx]
			entry.Running = false
			entry.Duration = msg.Duration
			entry.Failed = msg.Err != nil
		}

		// Refresh git stats after execution
		m.gitStats = detectGitStats(m.cwd)

		if msg.Err != nil {
			m.lastError = msg.Err
			m.breadcrumb.SetStatus("failed")
			m.router.SetLastCommand(m.router.LastCommand(), true) // Mark as failed

			m.appendLog("error", colors.BrightRed("Error: ")+msg.Err.Error())
			m.appendLog("info", colors.Dim("Tip: type 'again' or 'retry' to re-run"))
			m.appendLog("info", "")

			// Check for auto-fix
			if msg.Task != nil && m.retryCount == 0 {
				if fixTask := m.getFixTask(msg.Task); fixTask != nil {
					return m, func() tea.Msg {
						return AutofixStartMsg{
							OriginalTask: msg.Task,
							FixTask:      fixTask,
						}
					}
				}
			}
			m.state = StateIdle
			m.runLogIdx = -1
			return m, nil
		}

		m.breadcrumb.SetStatus("done")
		m.lastError = nil
		m.appendLog("info", "")
		m.state = StateIdle
		m.runLogIdx = -1
		return m, nil

	case AutofixStartMsg:
		m.state = StateAutofix
		m.breadcrumb.Push("fix")
		m.breadcrumb.SetStatus("auto-fixing...")

		m.runLogIdx = m.appendRunLog(msg.FixTask.Name + " (autofix)")

		return m, tea.Batch(m.spinner.Tick, m.runAutofixPipeline(msg.OriginalTask, msg.FixTask))

	case AutofixDoneMsg:
		if m.runLogIdx >= 0 && m.runLogIdx < len(m.log) {
			entry := &m.log[m.runLogIdx]
			entry.Running = false
			entry.Duration = msg.Duration
			entry.Failed = msg.Err != nil
		}

		if msg.Err != nil {
			m.lastError = msg.Err
			m.breadcrumb.SetStatus("fix failed")
			m.state = StateIdle
			m.runLogIdx = -1
			m.appendLog("error", colors.BrightRed("Autofix failed: ")+msg.Err.Error())
			return m, nil
		}
		// Fix succeeded, retry original task
		m.breadcrumb.Pop()
		m.breadcrumb.SetStatus("retrying...")
		m.retryCount++
		m.runLogIdx = -1
		return m, func() tea.Msg {
			return RetryMsg{Task: msg.OriginalTask}
		}

	case RetryMsg:
		m.state = StateRetrying
		m.runLogIdx = m.appendRunLog(msg.Task.Name + " (retry)")
		return m, tea.Batch(m.spinner.Tick, m.runPipeline(msg.Task))

	case ShellStartMsg:
		m.state = StateExecuting
		m.appendLog("prompt", "> "+msg.Command)
		return m, m.runShellCommand(msg.Command)

	case ShellDoneMsg:
		if output := strings.TrimRight(msg.Output, "\n"); output != "" {
			m.appendLog("output", output)
		}
		if msg.Err != nil {
			m.appendLog("error", fmt.Sprintf("%s (exit %d)",
				colors.BrightRed("failed"), msg.ExitCode))
			m.router.SetLastCommand(msg.Command, true) // Mark as failed
		}
		m.router.ShellHistory().Add(msg.Command, msg.ExitCode, msg.Duration, m.cwd)
		// Refresh git stats after shell command
		m.gitStats = detectGitStats(m.cwd)
		m.appendLog("info", "")
		m.state = StateIdle
		return m, nil
	}

	// Forward all other messages to the spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// handleKey processes keyboard input.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state != StateIdle {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return m, tea.Quit

	case "enter":
		return m.handleSubmit()

	case "backspace":
		if len(m.input) > 0 && m.cursor > 0 {
			m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
			m.cursor--
			// Update prompt mode when input changes
			m.promptMode = DetectPromptMode(m.input)
		}
		return m, nil

	case "delete":
		if m.cursor < len(m.input) {
			m.input = m.input[:m.cursor] + m.input[m.cursor+1:]
			m.promptMode = DetectPromptMode(m.input)
		}
		return m, nil

	case "left":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "right":
		if m.cursor < len(m.input) {
			m.cursor++
		}
		return m, nil

	case "home", "ctrl+a":
		m.cursor = 0
		return m, nil

	case "end", "ctrl+e":
		m.cursor = len(m.input)
		return m, nil

	case "up":
		return m.historyPrev(), nil

	case "down":
		return m.historyNext(), nil

	case "ctrl+u":
		m.input = m.input[m.cursor:]
		m.cursor = 0
		m.promptMode = DetectPromptMode(m.input)
		return m, nil

	case "ctrl+k":
		m.input = m.input[:m.cursor]
		m.promptMode = DetectPromptMode(m.input)
		return m, nil

	case "pgup":
		logHeight := m.logHeight()
		m.scrollOff += logHeight / 2
		maxScroll := len(m.log) - logHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scrollOff > maxScroll {
			m.scrollOff = maxScroll
		}
		return m, nil

	case "pgdown":
		logHeight := m.logHeight()
		m.scrollOff -= logHeight / 2
		if m.scrollOff < 0 {
			m.scrollOff = 0
		}
		return m, nil

	case "ctrl+l":
		m.log = m.log[:0]
		m.scrollOff = 0
		return m, nil

	default:
		if text := msg.Key().Text; text != "" {
			m.input = m.input[:m.cursor] + text + m.input[m.cursor:]
			m.cursor += len(text)
			// Update prompt mode when input changes
			m.promptMode = DetectPromptMode(m.input)
		}
		return m, nil
	}
}

// handleSubmit processes the entered command using the centralizedrouter.Router.
func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input)
	if input == "" {
		return m, nil
	}

	// Handle shell mode: strip $ prefix and route as shell command
	shellMode := m.promptMode == PromptModeShell
	if shellMode && len(input) > 0 && input[0] == '$' {
		input = strings.TrimSpace(input[1:])
	}

	m.input = ""
	m.cursor = 0
	m.promptMode = PromptModeLanguage

	// If in shell mode, directly execute as shell command
	if shellMode && input != "" {
		m.router.SetLastCommand(input, false)
		return m, func() tea.Msg {
			return ShellStartMsg{Command: input}
		}
	}

	// Handle pending confirmation (fuzzy match)
	if m.pendingConfirm != nil {
		confirm := m.pendingConfirm
		m.pendingConfirm = nil

		lower := strings.ToLower(input)
		if lower == "y" || lower == "yes" {
			// User confirmed, run the suggested task
			if resolved, err := m.agent.Resolver().Resolve(confirm.Suggestion); err == nil {
				m.router.SetLastCommand(confirm.Suggestion, false)
				return m, func() tea.Msg {
					return ExecutionStartMsg{
						Input:    confirm.Suggestion,
						Task:     resolved.Name,
						Resolved: resolved,
					}
				}
			}
		}
		// User declined or invalid response
		m.appendLog("info", colors.Dim("Cancelled"))
		m.appendLog("info", "")
		return m, nil
	}

	m.history = append(m.history, input)
	m.historyIdx = len(m.history)

	//router.Route input using centralized router (follows structure.d2 flow)
	route := m.router.Route(input)

	switch route.Type {
	case router.RouteQuit:
		return m, tea.Quit

	case router.RouteRetry:
		// Retry the last command
		lastCmd := m.router.LastCommand()
		if lastCmd == "" {
			m.appendLog("prompt", "> "+input)
			m.appendLog("error", "No previous command to retry")
			return m, nil
		}
		m.appendLog("prompt", "> "+input)
		m.appendLog("info", colors.Dim("Retrying: ")+lastCmd)
		// Re-route the last command
		retryRoute := m.router.Route(lastCmd)
		if retryRoute.Type == router.RouteTask || retryRoute.Type == router.RouteAlias {
			return m, func() tea.Msg {
				return ExecutionStartMsg{
					Input:    lastCmd,
					Task:     retryRoute.Task,
					Resolved: retryRoute.Resolved,
				}
			}
		} else if retryRoute.Type == router.RouteShell {
			return m, func() tea.Msg {
				return ShellStartMsg{Command: retryRoute.ShellCmd}
			}
		}
		m.appendLog("error", "Cannot retry: "+lastCmd)
		return m, nil

	case router.RouteConfirm:
		// Fuzzy match needs confirmation
		m.appendLog("prompt", "> "+input)
		m.pendingConfirm = route
		m.appendLog("info", fmt.Sprintf("Did you mean %s? [y/n]",
			colors.BrightGreen(route.Suggestion)))
		return m, nil

	case router.RouteSlash:
		m.appendLog("prompt", "> "+input)
		slashCmd, ok := m.registry.Get(route.Command)
		if ok && slashCmd != nil {
			return slashCmd.Handler(&m, route.Args)
		}
		m.appendLog("error", "Unknown command: /"+route.Command)
		m.appendLog("info", "")
		return m, nil

	case router.RouteMultiTask:
		// Run multiple tasks in sequence
		if len(route.Tasks) == 0 {
			m.appendLog("prompt", "> "+input)
			m.appendLog("error", "No tasks to run")
			return m, nil
		}
		// Start with the first task, chain the rest
		m.router.SetLastCommand(input, false)
		return m, func() tea.Msg {
			return ExecutionStartMsg{
				Input:    input,
				Task:     route.Tasks[0].Name,
				Resolved: route.Tasks[0],
			}
		}

	case router.RouteTask, router.RouteAlias:
		if route.Resolved == nil {
			m.appendLog("prompt", "> "+input)
			m.appendLog("error", "Could not resolve: "+input)
			m.appendLog("info", "")
			return m, nil
		}
		m.appendLog("prompt", "> "+input)
		m.router.SetLastCommand(input, false)
		return m, func() tea.Msg {
			return ExecutionStartMsg{
				Input:    input,
				Task:     route.Task,
				Resolved: route.Resolved,
			}
		}

	case router.RouteHelp:
		m.appendLog("prompt", "> "+input)
		m.appendLog("info", m.registry.HelpText())
		m.appendLog("info", "")
		return m, nil

	case router.RouteCorrection:
		m.router.Aliases().Add(route.Phrase, route.AliasTask)
		m.appendLog("prompt", "> "+input)
		m.appendLog("info", "Got it! \""+route.Phrase+"\" will now run "+colors.BrightGreen(route.AliasTask))
		m.appendLog("info", "")
		return m, nil

	case router.RouteGreeting:
		m.appendLog("prompt", "> "+input)
		m.appendLog("info", route.Greeting)
		m.appendLog("info", "")
		return m, nil

	case router.RouteFortune:
		m.appendLog("prompt", "> "+input)
		m.appendLog("info", route.Fortune)
		m.appendLog("info", "")
		return m, nil

	case router.RouteShell:
		m.router.SetLastCommand(input, false)
		return m, func() tea.Msg {
			return ShellStartMsg{Command: route.ShellCmd}
		}

	default:
		//router.RouteUnknown - show suggestions if ambiguous
		if route.Ambiguous {
			m.appendLog("prompt", "> "+input)
			var b strings.Builder
			if len(route.Matches) > 0 {
				b.WriteString("Matching skills:\n")
				for _, match := range route.Matches {
					b.WriteString("  " + colors.BrightGreen(match) + "\n")
				}
			}
			if len(route.HistMatches) > 0 {
				b.WriteString("From shell history:\n")
				for _, h := range route.HistMatches {
					status := colors.BrightGreen("exit 0")
					if h.ExitCode != 0 {
						status = colors.BrightRed(fmt.Sprintf("exit %d", h.ExitCode))
					}
					b.WriteString(fmt.Sprintf("  %s %s %s\n",
						colors.Dim("$"),
						h.Command,
						colors.Dim("("+status+")"),
					))
				}
			}
			b.WriteString("\nBe more specific or use /run <task>")
			m.appendLog("info", b.String())
			m.appendLog("info", "")
		} else {
			m.appendLog("prompt", "> "+input)
			m.appendLog("error", "Unknown command: "+input)
			m.appendLog("info", "")
		}
		return m, nil
	}
}
