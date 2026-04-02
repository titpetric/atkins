package agent

import (
	"github.com/titpetric/atkins/agent/history"
	"github.com/titpetric/atkins/model"
)

// RouteType categorizes the routing decision.
type RouteType int

// RouteType constants following structure.d2 flow.
const (
	RouteUnknown    RouteType = iota
	RouteAlias                // Resolved via alias
	RouteSlash                // Slash command (explicit or natural language)
	RouteTask                 // Skill/task execution
	RouteMultiTask            // Multiple tasks (chained with && or "then")
	RouteShell                // Shell command execution
	RouteGreeting             // Greeting response
	RouteCorrection           // Store a correction/alias
	RouteFortune              // Fortune/motivation request
	RouteHelp                 // Help request
	RouteQuit                 // Exit request
	RouteRetry                // Retry last command (again/retry)
	RouteConfirm              // Fuzzy match needs confirmation
)

// Route represents a routing decision.
type Route struct {
	Type        RouteType
	Raw         string                      // Original input
	Task        string                      // Task name for RouteTask
	Resolved    *model.ResolvedTask         // Resolved task for RouteTask
	Command     string                      // Slash command name for RouteSlash
	Args        string                      // Arguments for RouteSlash
	ShellCmd    string                      // Shell command for RouteShell
	Greeting    string                      // Greeting response for RouteGreeting
	Fortune     string                      // Fortune text for RouteFortune
	Phrase      string                      // Phrase for RouteCorrection
	AliasTask   string                      // Alias target for RouteCorrection
	Ambiguous   bool                        // Multiple matches found
	Matches     []string                    // Matching skills when ambiguous
	HistMatches []history.ShellHistoryEntry // Shell history matches

	// Multi-task support (chained commands)
	Tasks []*model.ResolvedTask // Multiple tasks for RouteMultiTask

	// Fuzzy match confirmation
	Suggestion string // Suggested correction for RouteConfirm
	Original   string // Original input that was fuzzy matched
}
