package agent

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/colors"
)

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
		// Gotestsum-style running indicator
		return fmt.Sprintf(" %s %s",
			m.spinner.View(),
			colors.BrightWhite(entry.Task))
	}

	// Gotestsum-style pass/fail format
	durStr := formatJobDuration(entry.Duration)
	if entry.Failed {
		return fmt.Sprintf(" %s %s %s",
			colors.BrightRed("✗"),
			colors.BrightWhite(entry.Task),
			colors.Dim("("+durStr+")"))
	}
	return fmt.Sprintf(" %s %s %s",
		colors.BrightGreen("✓"),
		colors.BrightWhite(entry.Task),
		colors.Dim("("+durStr+")"))
}

func (m Model) renderFooter(w int) string {
	// Border color - slate/teal
	borderColor := "\033[38;5;66m" // slate/teal color
	reset := "\033[0m"

	// Build the label: ~/path (branch) [+10 -5]
	label := m.shortenPath(m.cwd)
	if m.gitBranch != "" {
		label += " (" + m.gitBranch + ")"
	}

	// Add git stats if there are changes
	if m.gitStats.Added > 0 || m.gitStats.Removed > 0 {
		statsStr := " "
		if m.gitStats.Added > 0 {
			statsStr += colors.BrightGreen(fmt.Sprintf("+%d", m.gitStats.Added))
		}
		if m.gitStats.Removed > 0 {
			if m.gitStats.Added > 0 {
				statsStr += " "
			}
			statsStr += colors.BrightRed(fmt.Sprintf("-%d", m.gitStats.Removed))
		}
		label += statsStr
	}

	// Top border with label
	topLabel := label
	topRemain := w - 7 - colors.VisualLength(topLabel)
	if topRemain < 1 {
		topRemain = 1
	}
	topLine := borderColor + "╭─── " + reset + topLabel + " " + borderColor + strings.Repeat("─", topRemain) + "╮" + reset

	// Input line
	var prompt, inputText string
	if m.state != StateIdle {
		prompt = borderColor + "│" + reset + "   "
		inputText = m.input[:m.cursor] + m.input[m.cursor:]
	} else if m.promptMode == PromptModeShell {
		// Shell mode: $ in deep orange, command text in bright white
		prompt = borderColor + "│" + reset + " " + colors.BrightOrange("$") + " "
		// Strip the leading $ from input for display (already shown in prompt)
		displayInput := m.input
		if len(displayInput) > 0 && displayInput[0] == '$' {
			displayInput = displayInput[1:]
			// Also strip leading space if present
			if len(displayInput) > 0 && displayInput[0] == ' ' {
				displayInput = displayInput[1:]
			}
		}
		cursorPos := m.cursor
		if m.cursor > 0 {
			cursorPos-- // Account for stripped $
		}
		if cursorPos > 0 && len(m.input) > 1 && m.input[1] == ' ' {
			cursorPos-- // Account for stripped space
		}
		if cursorPos < 0 {
			cursorPos = 0
		}
		if cursorPos > len(displayInput) {
			cursorPos = len(displayInput)
		}
		inputText = colors.BrightWhite(displayInput[:cursorPos]) + "█" + colors.BrightWhite(displayInput[cursorPos:])
	} else {
		// Language mode: > prompt
		prompt = borderColor + "│" + reset + " > "
		inputText = m.input[:m.cursor] + "█" + m.input[m.cursor:]
	}
	inputLen := colors.VisualLength(prompt) + colors.VisualLength(inputText)
	inputPad := w - inputLen - 1 // 1 for trailing │
	if inputPad < 0 {
		inputPad = 0
	}
	midLine := prompt + inputText + strings.Repeat(" ", inputPad) + borderColor + "│" + reset

	// Bottom border
	bottomRemain := w - 2 // 2 for ╰ and ╯
	if bottomRemain < 0 {
		bottomRemain = 0
	}
	botLine := borderColor + "╰" + strings.Repeat("─", bottomRemain) + "╯" + reset

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

func (m Model) logHeight() int {
	h := m.height - 4 // 1 header + 3 footer
	if h < 1 {
		h = 1
	}
	return h
}
