package agent

import (
	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/agent/view"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	d := &view.RenderData{
		Width:           m.width,
		Height:          m.height,
		Version:         m.version,
		Hostname:        m.hostname,
		Cwd:             m.cwd,
		GitBranch:       m.gitBranch,
		GitAdded:        m.gitStats.Added,
		GitRemoved:      m.gitStats.Removed,
		Log:             m.log,
		ScrollOff:       m.scrollOff,
		Spinner:         m.spinner,
		ProgressSpinner: m.progressSpinner,
		State:           int(m.state),
		Input:           m.input,
		Cursor:          m.cursor,
		PromptMode:      m.promptMode,
	}
	return view.Render(d)
}

func (m Model) logHeight() int {
	return view.LogHeight(m.height)
}
