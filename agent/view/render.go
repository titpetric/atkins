package view

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/colors"
)

// RenderData holds all data needed to render the TUI view.
type RenderData struct {
	Width           int
	Height          int
	Version         string
	Hostname        string
	Cwd             string
	GitBranch       string
	GitAdded        int
	GitRemoved      int
	Log             []LogEntry
	ScrollOff       int
	Spinner         spinner.Model
	ProgressSpinner spinner.Model
	State           int // 0=idle
	Input           string
	Cursor          int
	PromptMode      PromptMode
}

// Render produces the full TUI view.
func Render(d *RenderData) tea.View {
	if d.Width == 0 || d.Height == 0 {
		return tea.NewView("")
	}

	var b strings.Builder
	w := d.Width

	// === Header bar ===
	header := RenderHeader(w, d.Version, d.Hostname)
	b.WriteString(header)
	b.WriteString("\n")

	// === Message log ===
	logH := LogHeight(d.Height)
	lines := RenderLog(d.Spinner, d.ProgressSpinner, d.Log, w)

	// Apply scroll offset
	start := len(lines) - logH - d.ScrollOff
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
	footer := RenderFooter(d.PromptMode, w, d.GitAdded, d.GitRemoved, d.State, d.Cursor, d.Cwd, d.GitBranch, d.Input)
	b.WriteString(footer)

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// RenderHeader renders the top header bar.
func RenderHeader(w int, version, hostname string) string {
	left := " 🔧 atkins"
	if version != "" {
		left += " " + colors.Dim("v"+version)
	}
	right := ""
	if hostname != "" {
		right = hostname + " "
	}

	leftLen := colors.VisualLength(left)
	rightLen := colors.VisualLength(right)
	padding := w - leftLen - rightLen
	if padding < 1 {
		padding = 1
	}

	return "\033[7m" + left + strings.Repeat(" ", padding) + right + "\033[0m"
}

// RenderLog renders all log entries into lines.
func RenderLog(spin spinner.Model, progressSpin spinner.Model, log []LogEntry, width int) []string {
	borderColor := "\033[38;5;66m"
	reset := "\033[0m"

	var lines []string
	inJobBlock := false

	closeJobBlock := func() {
		if inJobBlock {
			lines = append(lines, " "+borderColor+"╰────────────────────────────────────────"+reset)
			inJobBlock = false
		}
	}

	for _, entry := range log {
		switch entry.Kind {
		case "run":
			// The run entry opens the job result block (like shell-cmd opens output)
			closeJobBlock()
			lines = append(lines, RenderRunEntry(spin, entry))
			lines = append(lines, " "+borderColor+"╭────────────────────────────────────────"+reset)
			inJobBlock = true
		case "job-result":
			lines = append(lines, " "+borderColor+"│"+reset+" "+entry.Text)
		case "job-error":
			for _, l := range strings.Split(strings.TrimSpace(entry.Text), "\n") {
				lines = append(lines, " "+borderColor+"│"+reset+"   "+colors.Dim(l))
			}
		case "prompt":
			closeJobBlock()
			lines = append(lines, " "+colors.BrightCyan(entry.Text))
		case "shell-cmd":
			closeJobBlock()
			lines = append(lines, " "+colors.BrightOrange(entry.Text))
			lines = append(lines, " "+borderColor+"╭────────────────────────────────────────"+reset)
		case "output":
			// Trim leading/trailing empty lines and expand tabs
			text := strings.TrimSpace(entry.Text)
			for _, l := range strings.Split(text, "\n") {
				l = expandTabs(l, 8)
				lines = append(lines, " "+borderColor+"│"+reset+" "+l)
			}
		case "welcome":
			closeJobBlock()
			lines = append(lines, RenderWelcomeBox(entry.Text, width)...)
		default:
			closeJobBlock()
			for _, l := range strings.Split(entry.Text, "\n") {
				lines = append(lines, " "+l)
			}
		}
	}
	// If still inside a job block (execution in progress), show progress spinner
	if inJobBlock {
		lines = append(lines, " "+borderColor+"│"+reset+" "+progressSpin.View())
	}
	return lines
}

// expandTabs expands tab characters to spaces with the given tab width.
func expandTabs(s string, tabWidth int) string {
	var result strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := tabWidth - (col % tabWidth)
			result.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else {
			result.WriteRune(r)
			col++
		}
	}
	return result.String()
}

// RenderRunEntry renders a single run log entry.
func RenderRunEntry(spin spinner.Model, entry LogEntry) string {
	if entry.Running {
		if entry.Progress != "" {
			return fmt.Sprintf(" %s %s",
				spin.View(),
				entry.Progress)
		}
		return fmt.Sprintf(" %s %s",
			spin.View(),
			colors.BrightWhite(entry.Task))
	}

	durStr := FormatJobDuration(entry.Duration)
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

// RenderFooter renders the 3-line footer (border + input + bottom border).
func RenderFooter(promptMode PromptMode, w, gitAdded, gitRemoved, state, cursor int, cwd, gitBranch, input string) string {
	borderColor := "\033[38;5;66m"
	reset := "\033[0m"

	// Build the label: ~/path (branch) [+10 -5]
	label := ShortenPath(cwd)
	if gitBranch != "" {
		label += " (" + gitBranch + ")"
	}

	if gitAdded > 0 || gitRemoved > 0 {
		statsStr := " "
		if gitAdded > 0 {
			statsStr += colors.BrightGreen(fmt.Sprintf("+%d", gitAdded))
		}
		if gitRemoved > 0 {
			if gitAdded > 0 {
				statsStr += " "
			}
			statsStr += colors.BrightRed(fmt.Sprintf("-%d", gitRemoved))
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
	if state != 0 {
		prompt = borderColor + "│" + reset + "   "
		inputText = input[:cursor] + input[cursor:]
	} else if promptMode == PromptModeShell {
		prompt = borderColor + "│" + reset + " " + colors.BrightOrange("$") + " "
		displayInput := input
		if len(displayInput) > 0 && displayInput[0] == '$' {
			displayInput = displayInput[1:]
			if len(displayInput) > 0 && displayInput[0] == ' ' {
				displayInput = displayInput[1:]
			}
		}
		cursorPos := cursor
		if cursor > 0 {
			cursorPos--
		}
		if cursorPos > 0 && len(input) > 1 && input[1] == ' ' {
			cursorPos--
		}
		if cursorPos < 0 {
			cursorPos = 0
		}
		if cursorPos > len(displayInput) {
			cursorPos = len(displayInput)
		}
		inputText = colors.BrightWhite(displayInput[:cursorPos]) + "█" + colors.BrightWhite(displayInput[cursorPos:])
	} else {
		prompt = borderColor + "│" + reset + " > "
		inputText = input[:cursor] + "█" + input[cursor:]
	}
	inputLen := colors.VisualLength(prompt) + colors.VisualLength(inputText)
	inputPad := w - inputLen - 1
	if inputPad < 0 {
		inputPad = 0
	}
	midLine := prompt + inputText + strings.Repeat(" ", inputPad) + borderColor + "│" + reset

	// Bottom border
	bottomRemain := w - 2
	if bottomRemain < 0 {
		bottomRemain = 0
	}
	botLine := borderColor + "╰" + strings.Repeat("─", bottomRemain) + "╯" + reset

	return topLine + "\n" + midLine + "\n" + botLine
}

// RenderWelcomeBox renders the welcome message in a bordered box.
func RenderWelcomeBox(text string, width int) []string {
	borderColor := "\033[38;5;66m"
	reset := "\033[0m"

	contentLines := strings.Split(text, "\n")

	// Calculate inner width (leave room for borders and padding)
	innerW := width - 4 // 2 for border chars, 2 for padding
	if innerW < 10 {
		innerW = 10
	}

	var lines []string

	// Top border
	topBorder := borderColor + "╭" + strings.Repeat("─", width-2) + "╮" + reset
	lines = append(lines, topBorder)

	// Content lines with border
	for _, l := range contentLines {
		visLen := colors.VisualLength(l)
		pad := innerW - visLen
		if pad < 0 {
			pad = 0
		}
		line := borderColor + "│" + reset + " " + l + strings.Repeat(" ", pad) + " " + borderColor + "│" + reset
		lines = append(lines, line)
	}

	// Bottom border
	botBorder := borderColor + "╰" + strings.Repeat("─", width-2) + "╯" + reset
	lines = append(lines, botBorder)

	return lines
}

// ShortenPath replaces the home directory prefix with ~.
func ShortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// LogHeight calculates the available log area height.
func LogHeight(totalHeight int) int {
	h := totalHeight - 4 // 1 header + 3 footer
	if h < 1 {
		h = 1
	}
	return h
}
