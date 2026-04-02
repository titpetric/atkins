package agent

import tea "charm.land/bubbletea/v2"

// SlashCommand represents a slash command handler.
type SlashCommand struct {
	Name        string
	Aliases     []string
	Description string
	Handler     func(m *Model, args string) (Model, tea.Cmd)
}
