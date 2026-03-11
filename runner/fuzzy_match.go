package runner

import (
	"strings"

	"github.com/titpetric/atkins/model"
)

// FuzzyMatchError is returned when multiple fuzzy matches are found.
type FuzzyMatchError struct {
	Matches []*ResolvedTask
}

// Error returns the error as a user message.
func (e *FuzzyMatchError) Error() string {
	return "multiple jobs match the pattern; use -l to see all jobs"
}

// findFuzzyMatches finds all jobs matching the fuzzy pattern (suffix match).
// Priority order:
// 1. Exact match in main pipeline (no namespace) - highest priority
// 2. Exact matches in namespaced pipelines
// 3. Suffix matches
func findFuzzyMatches(pipelines []*model.Pipeline, pattern string) []*ResolvedTask {
	var mainExactMatches []*ResolvedTask
	var namespacedExactMatches []*ResolvedTask
	var suffixMatches []*ResolvedTask
	lowerPattern := strings.ToLower(pattern)

	for _, p := range pipelines {
		jobs := p.GetJobs()

		for jobName, job := range jobs {
			lowerJobName := strings.ToLower(jobName)
			match := NewResolvedTask(p, job, jobName)

			if lowerJobName == lowerPattern {
				if p.ID == "" {
					mainExactMatches = append(mainExactMatches, match)
				} else {
					namespacedExactMatches = append(namespacedExactMatches, match)
				}
			} else if strings.HasSuffix(lowerJobName, lowerPattern) {
				suffixMatches = append(suffixMatches, match)
			}
		}
	}

	if len(mainExactMatches) > 0 {
		return mainExactMatches
	}
	if len(namespacedExactMatches) > 0 {
		return namespacedExactMatches
	}
	return suffixMatches
}
