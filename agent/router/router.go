package router

import (
	"os/exec"
	"strings"

	"github.com/titpetric/atkins/agent/aliases"
	"github.com/titpetric/atkins/agent/greeting"
	"github.com/titpetric/atkins/agent/history"
	agentmodel "github.com/titpetric/atkins/agent/model"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// CommandLookup provides command existence checking.
type CommandLookup interface {
	HasCommand(name string) bool
}

// Router implements the centralized routing logic based on structure.d2.
// Flow: Prompt → Is alias? → Semantic parsing → Match (skills/targets)?
type Router struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline

	commands CommandLookup
	greeter  *greeting.Greeter
	aliases  *aliases.Aliases

	shellHistory *history.ShellHistory

	// Context for retry/again
	lastInput  string // Last input for retry
	lastFailed bool   // Whether last command failed
}

// NewRouter creates a new router with all dependencies.
func NewRouter(resolver *runner.TaskResolver, skills []*model.Pipeline, commands CommandLookup) *Router {
	return &Router{
		commands:     commands,
		greeter:      greeting.NewGreeter(),
		aliases:      aliases.NewAliasStore(),
		resolver:     resolver,
		skills:       skills,
		shellHistory: history.NewShellHistory(),
	}
}

// Aliases returns the alias store.
func (r *Router) Aliases() *aliases.Aliases {
	return r.aliases
}

// Greeter returns the greeter.
func (r *Router) Greeter() *greeting.Greeter {
	return r.greeter
}

// ShellHistory returns the shell history.
func (r *Router) ShellHistory() *history.ShellHistory {
	return r.shellHistory
}

// SetLastCommand records the last command for retry functionality.
func (r *Router) SetLastCommand(input string, failed bool) {
	r.lastInput = input
	r.lastFailed = failed
}

// LastCommand returns the last command input.
func (r *Router) LastCommand() string {
	return r.lastInput
}

// Route processes user input following the structure.d2 flow:
// 1. Is alias? → Replace with alias
// 2. Semantic parsing
// 3. Match prompt (skills & targets)?
//   - single match → execute
//   - multiple matches → ambiguous
//   - shell expression → shell exec
//   - greeting → greeting response
//   - store correction → save alias
//   - none → failure.
func (r *Router) Route(input string) *Route {
	input = strings.TrimSpace(input)
	if input == "" {
		return &Route{Type: RouteUnknown, Raw: input}
	}

	route := &Route{Raw: input}
	lower := strings.ToLower(input)

	// Step 0: Check for retry/again commands
	if lower == "again" || lower == "retry" || lower == "redo" {
		if r.lastInput != "" {
			route.Type = RouteRetry
			return route
		}
		route.Type = RouteUnknown
		return route
	}

	// Step 0.5: Check for command chaining (&&, "then", ";")
	if chainedRoute := r.parseChainedCommands(input); chainedRoute != nil {
		return chainedRoute
	}

	// Step 1: Check alias (Is alias? → yes → Replace with alias)
	if aliasTarget := r.aliases.Match(input); aliasTarget != "" {
		// Alias found - recursively route the target
		// But first check if it's a shell command
		if r.isShellCommand(aliasTarget) {
			route.Type = RouteShell
			route.ShellCmd = aliasTarget
			return route
		}
		// Try to resolve as task
		if resolved, err := r.resolver.Resolve(aliasTarget); err == nil {
			route.Type = RouteAlias
			route.Task = resolved.Name
			route.Resolved = resolved
			return route
		}
	}

	// Step 2: Check explicit slash command
	if strings.HasPrefix(input, "/") {
		return r.parseSlashCommand(input)
	}

	// Step 3: Check for quit/exit
	if lower == "quit" || lower == "exit" || lower == "q" {
		route.Type = RouteQuit
		return route
	}

	// Step 4: Check for help
	if lower == "help" || lower == "?" {
		route.Type = RouteHelp
		return route
	}

	// Step 5: Check for correction/alias definition
	if phrase, task, ok := aliases.ParseCorrection(input); ok {
		route.Type = RouteCorrection
		route.Phrase = phrase
		route.AliasTask = task
		return route
	}

	// Step 6: Check if teaching a greeting
	if word, learned := r.greeter.LearnGreeting(input); learned {
		route.Type = RouteGreeting
		route.Greeting = "Learned \"" + word + "\" as a greeting! Try it out."
		return route
	}

	// Step 7: Check for greeting
	if response := r.greeter.Match(input); response != "" {
		route.Type = RouteGreeting
		route.Greeting = response
		return route
	}

	// Step 8: Check for fortune
	if greeting.MatchFortune(input) {
		route.Type = RouteFortune
		route.Fortune = greeting.Fortune()
		return route
	}

	// Step 9: Check for natural language slash commands FIRST
	// "list" → /list, "list tasks" → /list, etc.
	// This takes precedence over shell commands for registered commands
	if slashRoute := r.matchNaturalSlashCommand(input); slashRoute != nil {
		return slashRoute
	}

	// Step 10: Check shell expression (command exists in PATH)
	// This prioritizes actual executables over natural language patterns
	// But slash commands take precedence (checked above)
	if r.isShellCommand(input) {
		route.Type = RouteShell
		route.ShellCmd = input
		return route
	}

	// Step 11: Check shell history for single match
	if histMatches := r.shellHistory.Match(input); len(histMatches) == 1 {
		cmd := histMatches[0].Command
		if r.isShellCommand(cmd) {
			route.Type = RouteShell
			route.ShellCmd = cmd
			return route
		}
	}

	// Step 12: Try natural language skill matching
	keywords := r.parseNaturalLanguage(input)
	if resolved := r.matchKeywordsToSkill(keywords); resolved != nil {
		route.Type = RouteTask
		route.Task = resolved.Name
		route.Resolved = resolved
		return route
	}

	// Step 13: Try direct task resolution
	if resolved, err := r.resolver.Resolve(input); err == nil {
		route.Type = RouteTask
		route.Task = resolved.Name
		route.Resolved = resolved
		return route
	}

	// Step 14: Show suggestions if partial matches exist
	skillMatches := r.FindMatches(keywords)
	histMatches := r.shellHistory.Match(input)

	if len(skillMatches) > 0 || len(histMatches) > 0 {
		route.Type = RouteUnknown
		route.Ambiguous = true
		route.Matches = skillMatches
		route.HistMatches = histMatches
		return route
	}

	// Step 15: Try fuzzy matching for typos
	if suggestion := r.fuzzyMatchSkill(input); suggestion != "" {
		route.Type = RouteConfirm
		route.Original = input
		route.Suggestion = suggestion
		return route
	}

	// No match found
	route.Type = RouteUnknown
	return route
}

// parseChainedCommands handles && and "then" for command chaining.
func (r *Router) parseChainedCommands(input string) *Route {
	// Check for && separator
	if strings.Contains(input, "&&") {
		parts := strings.Split(input, "&&")
		return r.resolveChain(parts)
	}

	// Check for " then " separator (natural language)
	if strings.Contains(strings.ToLower(input), " then ") {
		parts := strings.Split(strings.ToLower(input), " then ")
		return r.resolveChain(parts)
	}

	return nil
}

// resolveChain resolves multiple commands into a multi-task route.
func (r *Router) resolveChain(parts []string) *Route {
	var tasks []*model.ResolvedTask

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to resolve as task
		if resolved, err := r.resolver.Resolve(part); err == nil {
			tasks = append(tasks, resolved)
			continue
		}

		// Try natural language matching
		keywords := r.parseNaturalLanguage(part)
		if resolved := r.matchKeywordsToSkill(keywords); resolved != nil {
			tasks = append(tasks, resolved)
			continue
		}

		// If any part can't be resolved, fail the chain
		return nil
	}

	if len(tasks) < 2 {
		return nil
	}

	return &Route{
		Type:  RouteMultiTask,
		Tasks: tasks,
	}
}

// parseSlashCommand parses an explicit slash command.
func (r *Router) parseSlashCommand(input string) *Route {
	input = strings.TrimPrefix(input, "/")

	parts := strings.SplitN(input, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	route := &Route{
		Raw:     "/" + input,
		Command: command,
		Args:    args,
	}

	if command == "quit" || command == "exit" || command == "q" {
		route.Type = RouteQuit
		return route
	}
	if command == "help" || command == "h" || command == "?" {
		route.Type = RouteHelp
		return route
	}

	route.Type = RouteSlash
	return route
}

// matchNaturalSlashCommand matches natural language to slash commands.
// Examples: "list" → /list, "list tasks" → /list, "show skills" → /list
func (r *Router) matchNaturalSlashCommand(input string) *Route {
	lower := strings.ToLower(strings.TrimSpace(input))
	words := strings.Fields(lower)
	if len(words) == 0 {
		return nil
	}

	// Natural language patterns for slash commands
	patterns := map[string][]string{
		"list":    {"list", "list tasks", "list skills", "show tasks", "show skills", "tasks", "skills"},
		"help":    {"help", "help me", "what can you do", "commands"},
		"history": {"history", "show history", "command history"},
		"debug":   {"debug", "toggle debug"},
		"verbose": {"verbose", "toggle verbose"},
	}

	// Check each pattern
	for cmd, phrases := range patterns {
		for _, phrase := range phrases {
			if lower == phrase {
				// Verify the command exists in registry
				if r.commands != nil && r.commands.HasCommand(cmd) {
					return &Route{
						Type:    RouteSlash,
						Raw:     input,
						Command: cmd,
						Args:    "",
					}
				}
			}
		}
	}

	// Check if first word matches a slash command directly
	if r.commands != nil && r.commands.HasCommand(words[0]) {
		args := ""
		if len(words) > 1 {
			args = strings.Join(words[1:], " ")
		}
		return &Route{
			Type:    RouteSlash,
			Raw:     input,
			Command: words[0],
			Args:    args,
		}
	}

	return nil
}

// isShellCommand checks if the input is a valid shell command.
// Returns true if the first word is an executable in PATH.
func (r *Router) isShellCommand(input string) bool {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return false
	}

	// Check if first word is an executable
	// Note: We don't restrict to lowercase-only anymore to fix "curl wttr.in" issue
	if _, err := exec.LookPath(fields[0]); err == nil {
		return true
	}

	return false
}

// parseNaturalLanguage strips filler words and extracts core intent.
func (r *Router) parseNaturalLanguage(input string) []string {
	lower := strings.ToLower(input)

	// Remove punctuation
	lower = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' || r == ':' || r == '-' || r == '_' {
			return r
		}
		return ' '
	}, lower)

	words := strings.Fields(lower)

	fillerSet := make(map[string]bool)
	for _, f := range agentmodel.FillerWords {
		fillerSet[f] = true
	}

	var keywords []string
	for _, word := range words {
		if !fillerSet[word] && len(word) > 0 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// matchKeywordsToSkill tries to match extracted keywords to available skills.
func (r *Router) matchKeywordsToSkill(keywords []string) *model.ResolvedTask {
	if len(keywords) == 0 {
		return nil
	}

	allKW := agentmodel.ExpandKeywords(keywords)

	// 1. Try colon-joined combinations (most specific first)
	if len(allKW) >= 2 {
		for i := 0; i < len(keywords); i++ {
			for j := 0; j < len(allKW); j++ {
				if i == j {
					continue
				}
				combined := allKW[i] + ":" + allKW[j]
				if resolved, err := r.resolver.Resolve(combined); err == nil {
					return resolved
				}
			}
		}
	}

	// 2. Try each keyword as a direct task name
	for _, kw := range allKW {
		if resolved, err := r.resolver.Resolve(kw); err == nil {
			return resolved
		}
	}

	// 3. Try with :default suffix
	for _, kw := range allKW {
		if resolved, err := r.resolver.Resolve(kw + ":default"); err == nil {
			return resolved
		}
	}

	// 4. Match against job descriptions
	if resolved := r.matchByDescription(allKW); resolved != nil {
		return resolved
	}

	// 5. Loose prefix/substring matching
	if resolved := r.matchSkillLoose(allKW); resolved != nil {
		return resolved
	}

	return nil
}
