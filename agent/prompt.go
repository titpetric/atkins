package agent

// PromptMode represents the current input mode.
type PromptMode int

// PromptMode constants.
const (
	// PromptModeLanguage is the default mode for natural language input.
	// Display character: >
	PromptModeLanguage PromptMode = iota

	// PromptModeShell is the mode for shell command input.
	// Display character: $ (deep orange)
	// Triggered when input starts with $.
	PromptModeShell
)

// PromptChar returns the display character for the mode.
func (m PromptMode) PromptChar() string {
	switch m {
	case PromptModeShell:
		return "$"
	default:
		return ">"
	}
}

// DetectPromptMode returns the appropriate mode based on input.
// If input starts with "$", returns PromptModeShell.
// Otherwise returns PromptModeLanguage.
func DetectPromptMode(input string) PromptMode {
	if len(input) > 0 && input[0] == '$' {
		return PromptModeShell
	}
	return PromptModeLanguage
}
