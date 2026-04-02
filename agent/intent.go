package agent

import "github.com/titpetric/atkins/model"

// IntentType categorizes user input.
type IntentType int

// IntentType constants for the enum.
const (
	IntentUnknown IntentType = iota
	IntentTask               // Run a skill/task.
	IntentSlash              // Slash command.
	IntentHelp               // Help request.
	IntentQuit               // Exit request.
)

// Intent represents a parsed user intent.
type Intent struct {
	Type     IntentType
	Raw      string              // Original input
	Keywords []string            // Extracted keywords
	Task     string              // Resolved task name (e.g., "go:test")
	Command  string              // Slash command name (without /)
	Args     string              // Arguments for slash command
	Resolved *model.ResolvedTask // Resolved task reference
}
