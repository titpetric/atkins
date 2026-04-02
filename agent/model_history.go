package agent

// historyPrev moves to previous history entry.
func (m Model) historyPrev() Model {
	if len(m.history) == 0 {
		return m
	}
	if m.historyIdx > 0 {
		m.historyIdx--
		m.input = m.history[m.historyIdx]
		m.cursor = len(m.input)
		m.promptMode = DetectPromptMode(m.input)
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
		m.promptMode = DetectPromptMode(m.input)
	} else {
		m.historyIdx = len(m.history)
		m.input = ""
		m.cursor = 0
		m.promptMode = PromptModeLanguage
	}
	return m
}
