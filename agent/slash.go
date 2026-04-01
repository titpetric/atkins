package agent

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/colors"
)

// SlashCommand represents a slash command handler.
type SlashCommand struct {
	Name        string
	Aliases     []string
	Description string
	Handler     func(m *Model, args string) (Model, tea.Cmd)
}

// Registry holds all registered slash commands.
type Registry struct {
	commands map[string]*SlashCommand
	ordered  []string
}

// NewRegistry creates a new slash command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]*SlashCommand),
	}
}

// Register adds a slash command.
func (r *Registry) Register(cmd *SlashCommand) {
	r.commands[cmd.Name] = cmd
	r.ordered = append(r.ordered, cmd.Name)
	for _, alias := range cmd.Aliases {
		r.commands[alias] = cmd
	}
}

// Get retrieves a command by name or alias.
func (r *Registry) Get(name string) *SlashCommand {
	return r.commands[strings.ToLower(name)]
}

// HelpText returns formatted help text for all commands.
func (r *Registry) HelpText() string {
	var b strings.Builder
	b.WriteString("Available commands:\n\n")

	for _, name := range r.ordered {
		cmd := r.commands[name]
		if cmd.Description == "" {
			continue // hidden command
		}
		b.WriteString("  /")
		b.WriteString(cmd.Name)
		if len(cmd.Aliases) > 0 {
			b.WriteString(" (")
			for i, alias := range cmd.Aliases {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString("/")
				b.WriteString(alias)
			}
			b.WriteString(")")
		}
		b.WriteString("\n    ")
		b.WriteString(cmd.Description)
		b.WriteString("\n")
	}

	b.WriteString("\nYou can also type:\n")
	b.WriteString("  - Skill names directly: test, build, go:test\n")
	b.WriteString("  - Natural language: \"run the tests\", \"build it\"\n")

	return b.String()
}

// DefaultRegistry returns the built-in slash commands.
func DefaultRegistry() *Registry {
	r := NewRegistry()

	r.Register(&SlashCommand{
		Name:        "list",
		Description: "List available skills and jobs",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			pipelines := m.agent.Pipelines()
			if len(pipelines) == 0 {
				m.appendLog("info", "No skills available")
				return *m, nil
			}

			var lines []string
			for _, p := range pipelines {
				var prefix string
				if p.ID != "" {
					prefix = p.ID + ":"
				}

				jobNames := make([]string, 0, len(p.Jobs))
				for name := range p.Jobs {
					jobNames = append(jobNames, name)
				}
				sort.Strings(jobNames)

				for _, name := range jobNames {
					job := p.Jobs[name]
					fullName := prefix + name
					line := "  " + colors.BrightGreen(fullName)
					if job.Desc != "" {
						line += " - " + job.Desc
					}
					lines = append(lines, line)
				}
			}

			m.appendLog("info", "Available skills:\n\n"+strings.Join(lines, "\n"))
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "debug",
		Description: "Toggle debug mode",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			m.agent.options.Debug = !m.agent.options.Debug
			status := "off"
			if m.agent.options.Debug {
				status = "on"
			}
			m.appendLog("info", fmt.Sprintf("Debug mode: %s", status))
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "verbose",
		Aliases:     []string{"v"},
		Description: "Toggle verbose output",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			m.agent.options.Verbose = !m.agent.options.Verbose
			status := "off"
			if m.agent.options.Verbose {
				status = "on"
			}
			m.appendLog("info", fmt.Sprintf("Verbose mode: %s", status))
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "jail",
		Description: "Toggle jail mode (restrict to project scope)",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			m.agent.options.Jail = !m.agent.options.Jail
			status := "off"
			if m.agent.options.Jail {
				status = "on"
			}
			m.appendLog("info", fmt.Sprintf("Jail mode: %s", status))
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "help",
		Aliases:     []string{"h", "?"},
		Description: "Show this help message",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			m.appendLog("info", r.HelpText())
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "history",
		Description: "Show command history",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			if len(m.history) == 0 {
				m.appendLog("info", "No command history")
				return *m, nil
			}
			var lines []string
			for i, cmd := range m.history {
				lines = append(lines, fmt.Sprintf("  %d. %s", i+1, cmd))
			}
			m.appendLog("info", "Command history:\n\n"+strings.Join(lines, "\n"))
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "quit",
		Aliases:     []string{"q", "exit"},
		Description: "Exit the agent",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			return *m, tea.Quit
		},
	})

	r.Register(&SlashCommand{
		Name:        "run",
		Description: "Run a specific task (e.g., /run go:test)",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			args = strings.TrimSpace(args)
			if args == "" {
				m.appendLog("info", "Usage: /run <task>\nExample: /run go:test")
				return *m, nil
			}

			resolved, err := m.agent.Resolver().Resolve(args)
			if err != nil {
				m.appendLog("error", "Could not find task: "+args)
				return *m, nil
			}

			return *m, func() tea.Msg {
				return ExecutionStartMsg{
					Input:    "/run " + args,
					Task:     resolved.Name,
					Resolved: resolved,
				}
			}
		},
	})

	r.Register(&SlashCommand{
		Name:        "skills",
		Description: "Alias for /list - show available skills",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			return r.Get("list").Handler(m, args)
		},
	})

	r.Register(&SlashCommand{
		Name:        "cd",
		Description: "Change working directory (e.g., /cd .., /cd /path/to/dir)",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			args = strings.TrimSpace(args)
			if args == "" {
				m.appendLog("info", "Usage: /cd <path>\nCurrent: "+m.cwd)
				return *m, nil
			}

			if err := m.changeDir(args); err != nil {
				m.appendLog("error", "cd: "+err.Error())
				return *m, nil
			}

			m.appendLog("info", "Changed directory to "+m.cwd)
			return *m, nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "tree",
		Description: "Show available skills as a list",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			// In TUI mode, delegate to /list since treeview prints to stdout
			return r.Get("list").Handler(m, args)
		},
	})

	r.Register(&SlashCommand{
		Name:        "aliases",
		Description: "List defined aliases",
		Handler: func(m *Model, args string) (Model, tea.Cmd) {
			aliases := m.router.Aliases().Aliases
			if len(aliases) == 0 {
				m.appendLog("info", "No aliases defined.\n\nTeach an alias with:\n  alias <phrase> to <command>")
				return *m, nil
			}

			var lines []string
			for _, a := range aliases {
				lines = append(lines, fmt.Sprintf("  %s → %s",
					colors.BrightCyan(a.Phrase),
					colors.BrightGreen(a.Task)))
			}
			m.appendLog("info", "Defined aliases:\n\n"+strings.Join(lines, "\n"))
			return *m, nil
		},
	})

	return r
}
