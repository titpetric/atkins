package main

import (
	"strings"

	"github.com/titpetric/atkins/model"
)

// FuzzyMatch represents a fuzzy match result
type FuzzyMatch struct {
	Pipeline *model.Pipeline
	JobName  string
	FullName string // e.g., "skill:job" or just "job"
}

// FuzzyMatchError is returned when multiple fuzzy matches are found
type FuzzyMatchError struct {
	Matches []FuzzyMatch
}

func (e *FuzzyMatchError) Error() string {
	return "multiple jobs match the pattern; use -l to see all jobs"
}

// findFuzzyMatches finds all jobs matching the fuzzy pattern (suffix/substring match).
// Priority order:
// 1. Exact match in main pipeline (no namespace) - highest priority
// 2. Exact matches in namespaced pipelines
// 3. Fuzzy/substring matches
func findFuzzyMatches(pipelines []*model.Pipeline, pattern string) []FuzzyMatch {
	var mainExactMatches []FuzzyMatch       // Exact match in main pipeline (no ID)
	var namespacedExactMatches []FuzzyMatch // Exact match in namespaced pipelines
	var fuzzyMatches []FuzzyMatch
	lowerPattern := strings.ToLower(pattern)

	for _, p := range pipelines {
		jobs := p.Jobs
		if len(jobs) == 0 {
			jobs = p.Tasks
		}

		for jobName := range jobs {
			lowerJobName := strings.ToLower(jobName)
			fullName := jobName
			if p.ID != "" {
				fullName = p.ID + ":" + jobName
			}

			match := FuzzyMatch{
				Pipeline: p,
				JobName:  jobName,
				FullName: fullName,
			}

			// Check for exact match (case-insensitive)
			if lowerJobName == lowerPattern {
				if p.ID == "" {
					// Main pipeline exact match - highest priority
					mainExactMatches = append(mainExactMatches, match)
				} else {
					// Namespaced pipeline exact match
					namespacedExactMatches = append(namespacedExactMatches, match)
				}
			} else if strings.Contains(lowerJobName, lowerPattern) {
				fuzzyMatches = append(fuzzyMatches, match)
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
	return fuzzyMatches
}
