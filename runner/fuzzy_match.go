package runner

import (
	"strings"

	"github.com/titpetric/atkins/model"
)

// FuzzyMatchError is returned when multiple fuzzy matches are found.
type FuzzyMatchError struct {
	Matches []ResolvedTask
}

// Error returns the error as a user message.
func (e *FuzzyMatchError) Error() string {
	return "multiple jobs match the pattern; use -l to see all jobs"
}

// findFuzzyMatches finds all jobs matching the fuzzy pattern (suffix/substring match).
// 1. Exact match in main pipeline (no namespace) - highest priority
// 2. Exact matches in namespaced pipelines
// 3. Fuzzy/substring matches
func findFuzzyMatches(pipelines []*model.Pipeline, pattern string) []ResolvedTask {
	var mainExactMatches []ResolvedTask       // Exact match in main pipeline (no ID)
	var namespacedExactMatches []ResolvedTask // Exact match in namespaced pipelines
	var found []ResolvedTask
	lowerPattern := strings.ToLower(pattern)

	for _, p := range pipelines {
		jobs := p.GetJobs()

		for jobName, job := range jobs {
			lowerJobName := strings.ToLower(jobName)
			target := jobName

			match := ResolvedTask{
				Pipeline: p,
				Job:      job,
				Name:     target,
			}

			// Check for exact match (case-insensitive)
			if lowerJobName == lowerPattern {
				found = append(found, match)
			} else if strings.Contains(lowerJobName, lowerPattern) {
				found = append(found, match)
			}
		}
	}

	// Priority: main pipeline exact > namespaced exact > fuzzy
	if len(mainExactMatches) > 0 {
		return mainExactMatches
	}
	if len(namespacedExactMatches) > 0 {
		return namespacedExactMatches
	}
	return found
}
