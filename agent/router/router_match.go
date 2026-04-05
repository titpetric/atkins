package router

import (
	"strings"

	agentmodel "github.com/titpetric/atkins/agent/model"
	"github.com/titpetric/atkins/model"
)

// routerSkillInfo holds a skill name and its description for matching.
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

	allKW := agentmodel.ExpandKeywords(keywords)
	infos := r.allSkillInfos()
	seen := make(map[string]bool)
	var matches []string

	for _, kw := range allKW {
		// Skip very short keywords to avoid matching too many things
		if len(kw) < 2 {
			continue
		}
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
