package treeview

import (
	"slices"
)

// SortJobsByDepth returns the job names sorted by their count of ':', then by name.
func SortJobsByDepth(jobNames []string) []string {
	result := make([]string, len(jobNames))
	copy(result, jobNames)

	slices.SortFunc(result, compareByDepthThenName)
	return result
}

// compareByDepthThenName returns the comparison result for two job names.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
//
// Comparison order:
// 1. Sort by depth (count of ':' separators)
// 2. Within same depth, sort alphabetically by name
func compareByDepthThenName(a, b string) int {
	depthA := countDepth(a)
	depthB := countDepth(b)

	// Primary: compare by depth
	if depthA != depthB {
		if depthA < depthB {
			return -1
		}
		return 1
	}

	// Secondary: compare alphabetically
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// countDepth returns the depth of a job name based on ':' separators.
// For example:
//   - "test" -> 0
//   - "test:run" -> 1
//   - "test:run:subtask" -> 2
func countDepth(name string) int {
	count := 0
	for _, ch := range name {
		if ch == ':' {
			count++
		}
	}
	return count
}

// SortByOrder returns the set in orderList order, unlisted jobs last.
func SortByOrder(jobSet map[string]bool, orderList []string) []string {
	result := make([]string, 0, len(jobSet))

	// Add jobs in order from orderList
	for _, jobName := range orderList {
		if jobSet[jobName] {
			result = append(result, jobName)
		}
	}

	// Add any remaining jobs from the set not in orderList
	for jobName := range jobSet {
		found := false
		for _, ordered := range result {
			if ordered == jobName {
				found = true
				break
			}
		}
		if !found {
			result = append(result, jobName)
		}
	}

	return result
}
