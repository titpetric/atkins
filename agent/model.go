package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// State represents the current REPL state.
type State int

// State constants for the REPL.
const (
	StateIdle State = iota
	StateExecuting
	StateAutofix
	StateRetrying
)

// LogEntry represents a single entry in the message log.
type LogEntry struct {
	Time     time.Time
	Kind     string // "info", "error", "run", "prompt"
	Text     string
	Task     string
	Running  bool
	Started  time.Time
	Duration time.Duration
	Failed   bool
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

	// Intent parser and slash commands
	parser   *Parser
	registry *Registry

	// Dimensions
	width  int
	height int

	// TUI state
	version   string
	hostname  string
	cwd       string
	gitBranch string

	log          []LogEntry
	scrollOff    int
	spinner      spinner.Model
	runLogIdx    int // index of the current running entry in log
	shellHistory *ShellHistory
	greeter      *Greeter
}

// Messages for async operations.
type (
	ExecutionStartMsg struct {
		Input    string // original user input
		Task     string
		Resolved *model.ResolvedTask
	}
	ExecutionDoneMsg struct {
		Task     *model.ResolvedTask
		Err      error
		Duration time.Duration
	}
	AutofixStartMsg struct {
		OriginalTask *model.ResolvedTask
		FixTask      *model.ResolvedTask
	}
	AutofixDoneMsg struct {
		OriginalTask *model.ResolvedTask
		Err          error
		Duration     time.Duration
	}
	RetryMsg struct {
		Task *model.ResolvedTask
	}
	ShellStartMsg struct {
		Command string
	}
	ShellDoneMsg struct {
		Command  string
		Output   string
		Err      error
		ExitCode int
		Duration time.Duration
	}
)

// NewModel creates a new bubbletea model for the agent.
func NewModel(agent *Agent, version string) Model {
	cwd := agent.WorkDir()
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Model{
		agent:        agent,
		state:        StateIdle,
		history:      []string{},
		historyIdx:   -1,
		breadcrumb:   NewBreadcrumb(),
		parser:       NewParser(agent.Resolver(), agent.Pipelines()),
		registry:     DefaultRegistry(),
		version:      version,
		hostname:     detectHostname(),
		cwd:          cwd,
		gitBranch:    detectGitBranch(cwd),
		spinner:      s,
		runLogIdx:    -1,
		shellHistory: NewShellHistory(),
		greeter:      NewGreeter(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	m.appendGreeting()
	return m.spinner.Tick
}

func (m *Model) appendGreeting() {
	m.appendLog("info", colors.BrightCyan("Welcome to atkins.")+" Type a command to get started.")
	m.appendLog("info", colors.Dim("Commands:")+
		"  /help  /list  /run <task>  /cd <path>  /quit")

	// Show available targets inline
	skills := m.parser.AvailableSkills()
	if len(skills) > 0 {
		sort.Strings(skills)
		var names []string
		for i, s := range skills {
			if i >= 15 {
				names = append(names, fmt.Sprintf("... +%d more", len(skills)-i))
				break
			}
			names = append(names, colors.BrightGreen(s))
		}
		m.appendLog("info", colors.Dim("Targets:")+"  "+strings.Join(names, ", "))
	}
	m.appendLog("info", "")
}

func (m *Model) appendLog(kind, text string) {
	m.log = append(m.log, LogEntry{
		Time: time.Now(),
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
		Time:    time.Now(),
		Kind:    "run",
		Task:    task,
		Running: true,
		Started: time.Now(),
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

		m.appendLog("prompt", "> "+msg.Input)
		m.runLogIdx = m.appendRunLog(msg.Task)

		return m, tea.Batch(m.spinner.Tick, m.runPipeline(msg.Resolved))

	case ExecutionDoneMsg:
		if m.runLogIdx >= 0 && m.runLogIdx < len(m.log) {
			entry := &m.log[m.runLogIdx]
			entry.Running = false
			entry.Duration = msg.Duration
			entry.Failed = msg.Err != nil
		}

		if msg.Err != nil {
			m.lastError = msg.Err
			m.breadcrumb.SetStatus("failed")

			m.appendLog("error", colors.BrightRed("Error: ")+msg.Err.Error())
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
		}
		m.shellHistory.Add(msg.Command, msg.ExitCode, msg.Duration, m.cwd)
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
		}
		return m, nil

	case "delete":
		if m.cursor < len(m.input) {
			m.input = m.input[:m.cursor] + m.input[m.cursor+1:]
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
		return m, nil

	case "ctrl+k":
		m.input = m.input[:m.cursor]
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
		}
		return m, nil
	}
}

// handleSubmit processes the entered command.
func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input)
	if input == "" {
		return m, nil
	}

	m.history = append(m.history, input)
	m.historyIdx = len(m.history)
	m.input = ""
	m.cursor = 0

	// Parse intent
	intent, err := m.parser.Parse(input)
	if err != nil {
		m.appendLog("error", colors.BrightRed("Error: ")+err.Error())
		return m, nil
	}

	switch intent.Type {
	case IntentQuit:
		return m, tea.Quit

	case IntentSlash:
		m.appendLog("prompt", "> "+input)
		slashCmd := m.registry.Get(intent.Command)
		if slashCmd != nil {
			return slashCmd.Handler(&m, intent.Args)
		}
		m.appendLog("error", "Unknown command: /"+intent.Command)
		return m, nil

	case IntentTask:
		if intent.Resolved == nil {
			m.appendLog("prompt", "> "+input)
			m.appendLog("error", "Could not resolve: "+input)
			return m, nil
		}
		return m, func() tea.Msg {
			return ExecutionStartMsg{
				Input:    input,
				Task:     intent.Task,
				Resolved: intent.Resolved,
			}
		}

	case IntentHelp:
		m.appendLog("prompt", "> "+input)
		m.appendLog("info", m.registry.HelpText())
		return m, nil

	default:
		// Check if input is a correction/alias definition
		if phrase, task, ok := ParseCorrection(input); ok {
			m.parser.Aliases().Add(phrase, task)
			m.appendLog("prompt", "> "+input)
			m.appendLog("info", "Got it! \""+phrase+"\" will now run "+colors.BrightGreen(task))
			m.appendLog("info", "")
			return m, nil
		}

		// Check if input teaches a new greeting
		if word, learned := m.greeter.LearnGreeting(input); learned {
			m.appendLog("prompt", "> "+input)
			m.appendLog("info", "Learned \""+word+"\" as a greeting! Try it out.")
			m.appendLog("info", "")
			return m, nil
		}

		// Check if input is a greeting
		if response := m.greeter.Match(input); response != "" {
			m.appendLog("prompt", "> "+input)
			m.appendLog("info", response)
			m.appendLog("info", "")
			return m, nil
		}

		// Check if asking for a fortune
		if MatchFortune(input) {
			m.appendLog("prompt", "> "+input)
			m.appendLog("info", Fortune())
			m.appendLog("info", "")
			return m, nil
		}

		// Try shell fallback: check if the first word is an executable
		fields := strings.Fields(input)
		m.appendLog("info", fmt.Sprintf("input: %v", fields))

		if len(fields) > 0 && fields[0] == strings.ToLower(fields[0]) && fields[0] != "test" {
			if _, err := exec.LookPath(fields[0]); err == nil {
				return m, func() tea.Msg {
					return ShellStartMsg{Command: input}
				}
			}
		}

		// Check shell history for a matching command to re-run
		if histMatches := m.shellHistory.Match(input); len(histMatches) == 1 {
			cmd := histMatches[0].Command
			// Verify the binary still exists
			if hFields := strings.Fields(cmd); len(hFields) > 0 {
				if _, err := exec.LookPath(hFields[0]); err == nil {
					return m, func() tea.Msg {
						return ShellStartMsg{Command: cmd}
					}
				}
			}
		}

		/* Code from Parser.Parse(), needs updating to live here

		if strings.Contains(input, " ") {
			keywords := p.parseNaturalLanguage(input)
			intent.Keywords = keywords

			if resolved := p.matchKeywordsToSkill(keywords); resolved != nil {
				intent.Type = IntentTask
				intent.Task = resolved.Name
				intent.Resolved = resolved
				return intent, nil
			}
		} else {
			if taskName := p.aliases.Match(input); taskName != "" {
				if resolved, err := p.resolver.Resolve(taskName); err == nil {
					intent.Type = IntentTask
					intent.Task = resolved.Name
					intent.Resolved = resolved
					return intent, nil
				}
			}

			if resolved, err := p.resolver.Resolve(input); err == nil {
				intent.Type = IntentTask
				intent.Task = resolved.Name
				intent.Resolved = resolved
				return intent, nil
			}
		}
		*/

		// Show skill suggestions or shell history matches
		skillMatches := m.parser.FindMatches(intent.Keywords)
		histMatches := m.shellHistory.Match(input)

		if len(skillMatches) > 0 || len(histMatches) > 0 {
			m.appendLog("prompt", "> "+input)
			var b strings.Builder
			if len(skillMatches) > 0 {
				b.WriteString("Matching skills:\n")
				for _, match := range skillMatches {
					b.WriteString("  " + colors.BrightGreen(match) + "\n")
				}
			}
			if len(histMatches) > 0 {
				b.WriteString("From shell history:\n")
				for _, h := range histMatches {
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

// historyPrev moves to previous history entry.
func (m Model) historyPrev() Model {
	if len(m.history) == 0 {
		return m
	}
	if m.historyIdx > 0 {
		m.historyIdx--
		m.input = m.history[m.historyIdx]
		m.cursor = len(m.input)
	}
	return m
}

// historyNext moves to next history entry.
func (m Model) historyNext() Model {
	if len(m.history) == 0 {
		return m
	}
	if m.historyIdx < len(m.history)-1 {
		m.historyIdx++
		m.input = m.history[m.historyIdx]
		m.cursor = len(m.input)
	} else {
		m.historyIdx = len(m.history)
		m.input = ""
		m.cursor = 0
	}
	return m
}

func (m Model) logHeight() int {
	h := m.height - 4 // 1 header + 3 footer
	if h < 1 {
		h = 1
	}
	return h
}

// View implements tea.Model.
func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}

	var b strings.Builder
	w := m.width

	// === Header bar ===
	header := m.renderHeader(w)
	b.WriteString(header)
	b.WriteString("\n")

	// === Message log ===
	logH := m.logHeight()
	lines := m.renderLog(w)

	// Apply scroll offset
	start := len(lines) - logH - m.scrollOff
	if start < 0 {
		start = 0
	}
	end := start + logH
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]
	for _, line := range visible {
		b.WriteString(line)
		b.WriteString("\n")
	}
	// Fill remaining space
	for i := len(visible); i < logH; i++ {
		b.WriteString("\n")
	}

	// === Footer (3 lines) ===
	footer := m.renderFooter(w)
	b.WriteString(footer)

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m Model) renderHeader(w int) string {
	left := " 🔧 atkins"
	if m.version != "" {
		left += " " + colors.Dim("v"+m.version)
	}
	right := ""
	if m.hostname != "" {
		right = m.hostname + " "
	}

	leftLen := colors.VisualLength(left)
	rightLen := colors.VisualLength(right)
	padding := w - leftLen - rightLen
	if padding < 1 {
		padding = 1
	}

	return "\033[7m" + left + strings.Repeat(" ", padding) + right + "\033[0m"
}

func (m Model) renderLog(w int) []string {
	var lines []string
	for _, entry := range m.log {
		switch entry.Kind {
		case "run":
			lines = append(lines, m.renderRunEntry(entry))
		case "prompt":
			lines = append(lines, " "+colors.BrightCyan(entry.Text))
		case "output":
			for _, l := range strings.Split(entry.Text, "\n") {
				lines = append(lines, " "+colors.Dim("│")+" "+l)
			}
		default:
			for _, l := range strings.Split(entry.Text, "\n") {
				lines = append(lines, " "+l)
			}
		}
	}
	return lines
}

func (m Model) renderRunEntry(entry LogEntry) string {
	if entry.Running {
		return fmt.Sprintf(" %s Running %s...",
			m.spinner.View(),
			colors.BrightWhite(entry.Task))
	}

	dur := fmt.Sprintf("%.2fs", entry.Duration.Seconds())
	if entry.Failed {
		return fmt.Sprintf(" %s %s %s",
			colors.BrightRed("✗"),
			colors.BrightWhite(entry.Task),
			colors.BrightRed("FAIL")+" "+colors.Dim(dur))
	}
	return fmt.Sprintf(" %s %s %s",
		colors.BrightGreen("✓"),
		colors.BrightWhite(entry.Task),
		colors.BrightGreen("OK")+" "+colors.Dim(dur))
}

func (m Model) renderFooter(w int) string {
	// Build the label: ~/path (branch)
	label := m.shortenPath(m.cwd)
	if m.gitBranch != "" {
		label += " (" + m.gitBranch + ")"
	}

	// Top border with label
	topLabel := "─── " + label + " "
	topRemain := w - 2 - colors.VisualLength(topLabel) // 2 for ╭ and ╮
	if topRemain < 1 {
		topRemain = 1
	}
	topLine := "╭" + topLabel + strings.Repeat("─", topRemain) + "╮"

	// Input line
	prompt := "│ > "
	if m.state != StateIdle {
		prompt = "│   "
	}
	inputText := m.input[:m.cursor]
	if m.state == StateIdle {
		inputText += "█"
	}
	inputText += m.input[m.cursor:]
	inputLen := colors.VisualLength(prompt) + colors.VisualLength(inputText)
	inputPad := w - inputLen - 1 // 1 for trailing │
	if inputPad < 0 {
		inputPad = 0
	}
	midLine := prompt + inputText + strings.Repeat(" ", inputPad) + "│"

	// Bottom border
	bottomRemain := w - 2 // 2 for ╰ and ╯
	if bottomRemain < 0 {
		bottomRemain = 0
	}
	botLine := "╰" + strings.Repeat("─", bottomRemain) + "╯"

	return topLine + "\n" + midLine + "\n" + botLine
}

func (m Model) shortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// getFixTask returns the fix task for a given task, or nil if none exists.
func (m Model) getFixTask(task *model.ResolvedTask) *model.ResolvedTask {
	if task == nil || task.Pipeline == nil || task.Pipeline.ID == "" {
		return nil
	}
	fixName := task.Pipeline.ID + ":fix"
	fixTask, err := m.agent.Resolver().Resolve(fixName)
	if err != nil {
		return nil
	}
	return fixTask
}

// runPipeline executes the task silently and returns the result with duration.
func (m Model) runPipeline(task *model.ResolvedTask) tea.Cmd {
	return func() tea.Msg {
		jobName := task.Job.Name
		start := time.Now()

		ctx := context.Background()
		err := runner.RunPipeline(ctx, task.Pipeline, runner.PipelineOptions{
			Jobs:         []string{jobName},
			Silent:       true,
			Debug:        m.agent.Options().Debug,
			AllPipelines: m.agent.Pipelines(),
		})

		return ExecutionDoneMsg{
			Task:     task,
			Err:      err,
			Duration: time.Since(start),
		}
	}
}

// runAutofixPipeline runs the fix task and then signals completion.
func (m Model) runAutofixPipeline(originalTask, fixTask *model.ResolvedTask) tea.Cmd {
	return func() tea.Msg {
		jobName := fixTask.Job.Name
		start := time.Now()

		ctx := context.Background()
		err := runner.RunPipeline(ctx, fixTask.Pipeline, runner.PipelineOptions{
			Jobs:         []string{jobName},
			Silent:       true,
			Debug:        m.agent.Options().Debug,
			AllPipelines: m.agent.Pipelines(),
		})

		return AutofixDoneMsg{
			OriginalTask: originalTask,
			Err:          err,
			Duration:     time.Since(start),
		}
	}
}

// runShellCommand runs a shell command and captures output.
func (m Model) runShellCommand(command string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = m.cwd
		out, err := cmd.CombinedOutput()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		return ShellDoneMsg{
			Command:  command,
			Output:   string(out),
			Err:      err,
			ExitCode: exitCode,
			Duration: time.Since(start),
		}
	}
}

// Helper functions for hostname and git branch detection.

func detectHostname() string {
	out, err := exec.Command("uname", "-n").Output()
	if err != nil {
		if h, err := os.Hostname(); err == nil {
			return h
		}
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectGitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// refreshCwd updates the current working directory and git branch.
func (m *Model) refreshCwd() {
	if cwd, err := os.Getwd(); err == nil {
		m.cwd = cwd
	}
	m.gitBranch = detectGitBranch(m.cwd)
}

// changeDir handles changing the working directory and reloading pipelines.
func (m *Model) changeDir(dir string) error {
	target := dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(m.cwd, target)
	}
	target = filepath.Clean(target)

	if err := os.Chdir(target); err != nil {
		return err
	}

	m.cwd = target
	m.gitBranch = detectGitBranch(target)
	m.agent.workDir = target

	// Reload pipelines for new directory
	loader := runner.NewSkillsLoader(target, target)
	pipelines, err := loader.Load()
	if err != nil {
		pipelines = []*model.Pipeline{}
	}

	if !m.agent.options.Jail {
		if home, err := os.UserHomeDir(); err == nil {
			globalLoader := runner.NewSkillsLoader(target, target)
			globalLoader.SkillsDirs = []string{filepath.Join(home, ".atkins", "skills")}
			if globalPipelines, globalErr := globalLoader.Load(); globalErr == nil {
				seen := make(map[string]bool)
				for _, p := range pipelines {
					if p.ID != "" {
						seen[p.ID] = true
					}
				}
				for _, gp := range globalPipelines {
					if !seen[gp.ID] {
						pipelines = append(pipelines, gp)
					}
				}
			}
		}
	}

	if configPath, _, err := runner.DiscoverConfigFromCwd(); err == nil && configPath != "" {
		if mainPipelines, loadErr := runner.LoadPipeline(configPath); loadErr == nil {
			pipelines = append(mainPipelines, pipelines...)
		}
	}

	m.agent.pipelines = pipelines
	m.agent.resolver = runner.NewTaskResolver(pipelines)
	m.parser = NewParser(m.agent.Resolver(), m.agent.Pipelines())

	return nil
}
