package runner

import "fmt"

// generateStepID creates a step ID from job name and sequential step index
// Format follows GitHub Actions: jobs.<jobName>.steps.<sequentialIndex>
func generateStepID(jobName string, stepIndex int) string {
	if jobName == "" {
		return ""
	}
	// Format: jobs.<jobName>.steps.<sequentialIndex>
	return "jobs." + jobName + ".steps." + fmt.Sprintf("%d", stepIndex)
}
