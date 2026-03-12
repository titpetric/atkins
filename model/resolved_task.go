package model

// ResolvedTask contains the result of resolving a task reference.
type ResolvedTask struct {
	Name     string    // Canonical name (e.g., "go:build" or "build")
	Job      *Job      // The resolved job
	Pipeline *Pipeline // The pipeline containing the task
}

// NewResolvedTask creates a ResolvedTask with all required fields.
func NewResolvedTask(pipeline *Pipeline, job *Job, name string) *ResolvedTask {
	return &ResolvedTask{
		Name:     name,
		Job:      job,
		Pipeline: pipeline,
	}
}
