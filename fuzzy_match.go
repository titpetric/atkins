package main

import (
	"fmt"
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
	return fmt.Sprintf("multiple jobs match the pattern; use -l to see all jobs")
}

// findFuzzyMatches finds all jobs matching the fuzzy pattern (suffix/substring match)
func findFuzzyMatches(pipelines []*model.Pipeline, pattern string) []FuzzyMatch {
	var matches []FuzzyMatch
	lowerPattern := strings.ToLower(pattern)

	for _, p := range pipelines {
		jobs := p.Jobs
		if len(jobs) == 0 {
			jobs = p.Tasks
		}

		for jobName := range jobs {
			if strings.Contains(strings.ToLower(jobName), lowerPattern) {
				fullName := jobName
				if p.ID != "" {
					fullName = p.ID + ":" + jobName
				}
				matches = append(matches, FuzzyMatch{
					Pipeline: p,
					JobName:  jobName,
					FullName: fullName,
				})
			}
		}
	}

	return matches
}
