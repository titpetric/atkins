package agent

import (
	"os/exec"
	"strings"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
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
	Raw         string              // Original input
	Task        string              // Task name for RouteTask
	Resolved    *model.ResolvedTask // Resolved task for RouteTask
	Command     string              // Slash command name for RouteSlash
	Args        string              // Arguments for RouteSlash
	ShellCmd    string              // Shell command for RouteShell
	Greeting    string              // Greeting response for RouteGreeting
	Fortune     string              // Fortune text for RouteFortune
	Phrase      string              // Phrase for RouteCorrection
	AliasTask   string              // Alias target for RouteCorrection
	Ambiguous   bool                // Multiple matches found
	Matches     []string            // Matching skills when ambiguous
	HistMatches []ShellHistoryEntry // Shell history matches

	// Multi-task support (chained commands)
	Tasks []*model.ResolvedTask // Multiple tasks for RouteMultiTask

	// Fuzzy match confirmation
	Suggestion string // Suggested correction for RouteConfirm
	Original   string // Original input that was fuzzy matched
}

// Router implements the centralized routing logic based on structure.d2.
// Flow: Prompt → Is alias? → Semantic parsing → Match (skills/targets)?
type Router struct {
	resolver     *runner.TaskResolver
	skills       []*model.Pipeline
	aliases      *AliasStore
	greeter      *Greeter
	registry     *Registry
	shellHistory *ShellHistory

	// Context for retry/again
	lastInput  string // Last input for retry
	lastFailed bool   // Whether last command failed
}

// NewRouter creates a new router with all dependencies.
func NewRouter(resolver *runner.TaskResolver, skills []*model.Pipeline, registry *Registry) *Router {
	return &Router{
		resolver:     resolver,
		skills:       skills,
		aliases:      NewAliasStore(),
		greeter:      NewGreeter(),
		registry:     registry,
		shellHistory: NewShellHistory(),
	}
}

// Aliases returns the alias store.
func (r *Router) Aliases() *AliasStore {
	return r.aliases
}

// Greeter returns the greeter.
func (r *Router) Greeter() *Greeter {
	return r.greeter
}

// ShellHistory returns the shell history.
func (r *Router) ShellHistory() *ShellHistory {
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
//   - none → failure
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
	if phrase, task, ok := ParseCorrection(input); ok {
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
	if MatchFortune(input) {
		route.Type = RouteFortune
		route.Fortune = Fortune()
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

// fuzzyMatchSkill attempts to find a close match for typos.
func (r *Router) fuzzyMatchSkill(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if len(input) < 2 {
		return ""
	}

	infos := r.allSkillInfos()
	var bestMatch string
	bestScore := 0

	for _, info := range infos {
		name := strings.ToLower(info.name)

		// Calculate similarity score
		score := r.similarityScore(input, name)

		// Also check against the job name part (after colon)
		if idx := strings.LastIndex(name, ":"); idx >= 0 {
			jobName := name[idx+1:]
			jobScore := r.similarityScore(input, jobName)
			if jobScore > score {
				score = jobScore
			}
		}

		if score > bestScore && score >= 60 { // 60% similarity threshold
			bestScore = score
			bestMatch = info.name
		}
	}

	return bestMatch
}

// similarityScore calculates a simple similarity score (0-100).
func (r *Router) similarityScore(a, b string) int {
	if a == b {
		return 100
	}

	// Check prefix match
	if strings.HasPrefix(b, a) || strings.HasPrefix(a, b) {
		shorter := len(a)
		if len(b) < shorter {
			shorter = len(b)
		}
		longer := len(a)
		if len(b) > longer {
			longer = len(b)
		}
		return (shorter * 100) / longer
	}

	// Simple Levenshtein-inspired score
	dist := levenshteinDistance(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 0
	}

	return ((maxLen - dist) * 100) / maxLen
}

// levenshteinDistance calculates edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
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
				if r.registry != nil && r.registry.Get(cmd) != nil {
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
	if r.registry != nil && r.registry.Get(words[0]) != nil {
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
	for _, f := range FillerWords {
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

	allKW := expandKeywords(keywords)

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

// skillInfo holds a skill name and its description for matching.
type routerSkillInfo struct {
	name string
	desc string
}

// allSkillInfos returns all available skills with descriptions.
func (r *Router) allSkillInfos() []routerSkillInfo {
	var infos []routerSkillInfo
	seen := make(map[string]bool)

	for _, pipeline := range r.skills {
		for name, job := range pipeline.Jobs {
			var fullName string
			if pipeline.ID != "" {
				fullName = pipeline.ID + ":" + name
			} else {
				fullName = name
			}
			if !seen[fullName] {
				seen[fullName] = true
				infos = append(infos, routerSkillInfo{name: fullName, desc: job.Desc})
			}
		}
	}

	return infos
}

// matchByDescription scores skills by how many keywords appear in their
// name or description. Returns the best match if unambiguous.
func (r *Router) matchByDescription(keywords []string) *model.ResolvedTask {
	type scored struct {
		name  string
		score int
	}

	infos := r.allSkillInfos()
	var results []scored

	for _, info := range infos {
		nameLower := strings.ToLower(info.name)
		descLower := strings.ToLower(info.desc)
		score := 0

		for _, kw := range keywords {
			if strings.Contains(nameLower, kw) {
				score += 2
			}
			if descLower != "" && strings.Contains(descLower, kw) {
				score++
			}
		}
		if score > 0 {
			results = append(results, scored{name: info.name, score: score})
		}
	}

	if len(results) == 0 {
		return nil
	}

	// Find best score
	best := results[0]
	for _, r := range results[1:] {
		if r.score > best.score {
			best = r
		}
	}

	// Only auto-resolve if clear winner
	secondBest := 0
	for _, r := range results {
		if r.name != best.name && r.score > secondBest {
			secondBest = r.score
		}
	}

	if best.score > secondBest {
		if resolved, err := r.resolver.Resolve(best.name); err == nil {
			return resolved
		}
	}

	return nil
}

// matchSkillLoose does prefix and substring matching.
func (r *Router) matchSkillLoose(keywords []string) *model.ResolvedTask {
	infos := r.allSkillInfos()

	type scored struct {
		name  string
		hits  int
		parts int
	}

	var candidates []scored
	for _, info := range infos {
		nameLower := strings.ToLower(info.name)
		hits := 0
		for _, kw := range keywords {
			if strings.HasPrefix(nameLower, kw+":") || strings.Contains(nameLower, kw) {
				hits++
			}
		}
		if hits > 0 {
			parts := strings.Count(info.name, ":") + 1
			candidates = append(candidates, scored{name: info.name, hits: hits, parts: parts})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Find best
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.hits > best.hits || (c.hits == best.hits && c.parts > best.parts) {
			best = c
		}
	}

	// Check for clear winner
	if len(candidates) == 1 || best.hits > candidates[1].hits {
		if resolved, err := r.resolver.Resolve(best.name); err == nil {
			return resolved
		}
	}

	return nil
}

// FindMatches returns all skills matching the input keywords.
func (r *Router) FindMatches(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}

	allKW := expandKeywords(keywords)
	infos := r.allSkillInfos()
	seen := make(map[string]bool)
	var matches []string

	for _, kw := range allKW {
		for _, info := range infos {
			if seen[info.name] {
				continue
			}
			nameLower := strings.ToLower(info.name)
			descLower := strings.ToLower(info.desc)
			if strings.Contains(nameLower, kw) ||
				strings.HasPrefix(nameLower, kw+":") ||
				(descLower != "" && strings.Contains(descLower, kw)) {
				seen[info.name] = true
				matches = append(matches, info.name)
			}
		}
	}

	return matches
}

// AvailableSkills returns a list of available skill names.
func (r *Router) AvailableSkills() []string {
	infos := r.allSkillInfos()
	skills := make([]string, len(infos))
	for i, info := range infos {
		skills[i] = info.name
	}
	return skills
}
