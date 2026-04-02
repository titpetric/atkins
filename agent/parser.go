package agent

import (
	"sort"
	"strings"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// FillerWords to strip from natural language input.
var FillerWords = []string{
	"give", "me", "the", "a", "an", "please", "can", "you",
	"i", "want", "need", "get", "show", "run", "execute",
	"do", "make", "let", "lets", "let's", "my", "some",
	"what", "is", "are", "how", "about", "whats", "what's",
	"your", "its", "it's", "tell", "whats",
}

// Parser parses user input into intents.
type Parser struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline
	aliases  *AliasStore
}

// NewParser creates a new intent parser.
func NewParser(resolver *runner.TaskResolver, skills []*model.Pipeline) *Parser {
	return &Parser{
		resolver: resolver,
		skills:   skills,
		aliases:  NewAliasStore(),
	}
}

// Parse analyzes input and returns an Intent.
func (p *Parser) Parse(input string) (*Intent, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return &Intent{Type: IntentUnknown, Raw: input}, nil
	}

	intent := &Intent{Raw: input}

	// Check for slash command
	if strings.HasPrefix(input, "/") {
		return p.parseSlashCommand(input)
	}

	// Check for quit commands
	lower := strings.ToLower(input)
	if lower == "quit" || lower == "exit" || lower == "q" {
		intent.Type = IntentQuit
		return intent, nil
	}

	// Check for help commands
	if lower == "help" || lower == "?" {
		intent.Type = IntentHelp
		return intent, nil
	}

	// Could not resolve
	intent.Type = IntentUnknown
	return intent, nil
}

// Aliases returns the alias store for external use.
func (p *Parser) Aliases() *AliasStore {
	return p.aliases
}

// parseSlashCommand parses a slash command.
func (p *Parser) parseSlashCommand(input string) (*Intent, error) {
	input = strings.TrimPrefix(input, "/")

	parts := strings.SplitN(input, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	if command == "quit" || command == "exit" || command == "q" {
		return &Intent{Type: IntentQuit, Command: command}, nil
	}
	if command == "help" || command == "h" || command == "?" {
		return &Intent{Type: IntentHelp, Command: command}, nil
	}

	return &Intent{
		Type:    IntentSlash,
		Raw:     input,
		Command: command,
		Args:    args,
	}, nil
}

// parseNaturalLanguage strips filler words and extracts core intent.
func (p *Parser) parseNaturalLanguage(input string) []string {
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

// skillInfo holds a skill name and its description for matching.
type skillInfo struct {
	name string
	desc string
}

// allSkillInfos returns all available skills with descriptions.
func (p *Parser) allSkillInfos() []skillInfo {
	var infos []skillInfo
	seen := make(map[string]bool)

	for _, pipeline := range p.skills {
		for name, job := range pipeline.Jobs {
			var fullName string
			if pipeline.ID != "" {
				fullName = pipeline.ID + ":" + name
			} else {
				fullName = name
			}
			if !seen[fullName] {
				seen[fullName] = true
				infos = append(infos, skillInfo{name: fullName, desc: job.Desc})
			}
		}
	}

	return infos
}

// singularize strips common plural suffixes.
func singularize(word string) string {
	if strings.HasSuffix(word, "ies") && len(word) > 3 {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ses") && len(word) > 3 {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && len(word) > 1 {
		return word[:len(word)-1]
	}
	return word
}

// expandKeywords returns the original keywords plus singularized variants.
func expandKeywords(keywords []string) []string {
	expanded := make([]string, 0, len(keywords)*2)
	seen := make(map[string]bool)
	for _, kw := range keywords {
		if !seen[kw] {
			expanded = append(expanded, kw)
			seen[kw] = true
		}
		s := singularize(kw)
		if s != kw && !seen[s] {
			expanded = append(expanded, s)
			seen[s] = true
		}
	}
	return expanded
}

// matchKeywordsToSkill tries to match extracted keywords to available skills.
func (p *Parser) matchKeywordsToSkill(keywords []string) *model.ResolvedTask {
	if len(keywords) == 0 {
		return nil
	}

	allKW := expandKeywords(keywords)

	// 1. Try colon-joined combinations (most specific first)
	//    ["go", "test"] → "go:test"
	//    ["test", "simple"] → "test:simple"
	if len(allKW) >= 2 {
		// Try all pairs, original keywords first
		for i := 0; i < len(keywords); i++ {
			for j := 0; j < len(allKW); j++ {
				if i == j {
					continue
				}
				combined := allKW[i] + ":" + allKW[j]
				if resolved, err := p.resolver.Resolve(combined); err == nil {
					return resolved
				}
			}
		}
	}

	// 2. Try each keyword as a direct task name
	for _, kw := range allKW {
		if resolved, err := p.resolver.Resolve(kw); err == nil {
			return resolved
		}
	}

	// 3. Try with :default suffix
	for _, kw := range allKW {
		if resolved, err := p.resolver.Resolve(kw + ":default"); err == nil {
			return resolved
		}
	}

	// 4. Match against job descriptions
	if resolved := p.matchByDescription(allKW); resolved != nil {
		return resolved
	}

	// 5. Loose prefix/substring matching — prefer more specific (longer) matches.
	if resolved := p.matchSkillLoose(allKW); resolved != nil {
		return resolved
	}

	return nil
}

// matchByDescription scores skills by how many keywords appear in their
// name or description. Returns the best match if unambiguous.
func (p *Parser) matchByDescription(keywords []string) *model.ResolvedTask {
	type scored struct {
		name  string
		score int
	}

	infos := p.allSkillInfos()
	var results []scored

	for _, info := range infos {
		nameLower := strings.ToLower(info.name)
		descLower := strings.ToLower(info.desc)
		score := 0

		for _, kw := range keywords {
			// Name match is worth more
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

	// Sort by score descending, then by name length ascending (prefer specific)
	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return len(results[i].name) > len(results[j].name)
	})

	// Only auto-resolve if there's a clear winner
	if len(results) == 1 || results[0].score > results[1].score {
		if resolved, err := p.resolver.Resolve(results[0].name); err == nil {
			return resolved
		}
	}

	return nil
}

// matchSkillLoose does prefix and substring matching of keywords against the
// full list of available skill names. Prefers more specific (longer name) matches.
// Returns a resolved task only when there is exactly one best match.
func (p *Parser) matchSkillLoose(keywords []string) *model.ResolvedTask {
	infos := p.allSkillInfos()

	// Score each skill: count how many keywords match
	type scored struct {
		name  string
		hits  int
		parts int // number of colon-separated parts (more = more specific)
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

	// Sort: most keyword hits first, then most specific (more parts), then longer name
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].hits != candidates[j].hits {
			return candidates[i].hits > candidates[j].hits
		}
		if candidates[i].parts != candidates[j].parts {
			return candidates[i].parts > candidates[j].parts
		}
		return len(candidates[i].name) > len(candidates[j].name)
	})

	// Only return if there's a single best or a clear winner
	if len(candidates) == 1 {
		if resolved, err := p.resolver.Resolve(candidates[0].name); err == nil {
			return resolved
		}
	}

	// Clear winner: top candidate has strictly more hits than second
	if candidates[0].hits > candidates[1].hits {
		if resolved, err := p.resolver.Resolve(candidates[0].name); err == nil {
			return resolved
		}
	}

	return nil
}

// FindMatches returns all skills matching the input keywords.
// Includes description matching.
func (p *Parser) FindMatches(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}

	allKW := expandKeywords(keywords)
	infos := p.allSkillInfos()
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

// AvailableSkills returns a list of available skill names for completion.
func (p *Parser) AvailableSkills() []string {
	infos := p.allSkillInfos()
	skills := make([]string, len(infos))
	for i, info := range infos {
		skills[i] = info.name
	}
	return skills
}
