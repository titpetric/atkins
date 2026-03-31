package model

import "sort"

// StepDispatcher handles dispatching step execution without caring
// about the step type (task vs command). Implementations decide how
// to execute each type.
type StepDispatcher interface {
	// Task is called when the step references another job via task:.
	Task(step *Step) error
	// Command is called for each executable command in the step.
	Command(step *Step, index int, cmd string) error
}

// PipelineWalkFunc is called for each job in a pipeline during Walk.
type PipelineWalkFunc func(name string, job *Job) error

// Walk iterates over all jobs in the pipeline in deterministic order.
// The "default" job is visited first, followed by remaining jobs in
// sorted order.
func (p *Pipeline) Walk(fn PipelineWalkFunc) error {
	jobs := p.GetJobs()
	keys := make([]string, 0, len(jobs))
	hasDefault := false

	for key := range jobs {
		if key == "default" {
			hasDefault = true
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if hasDefault {
		keys = append([]string{"default"}, keys...)
	}

	for _, key := range keys {
		if err := fn(key, jobs[key]); err != nil {
			return err
		}
	}
	return nil
}

// JobWalkFunc is called for each step in a job during Walk.
type JobWalkFunc func(index int, step *Step) error

// Walk iterates over all steps (children) of the job.
func (j *Job) Walk(fn JobWalkFunc) error {
	for i, step := range j.Children() {
		if err := fn(i, step); err != nil {
			return err
		}
	}
	return nil
}

// Dispatch routes the step to the appropriate StepDispatcher method
// based on the step type. Task steps call d.Task, command steps call
// d.Command for each command.
func (s *Step) Dispatch(d StepDispatcher) error {
	if s.Task != "" {
		return d.Task(s)
	}
	for i, cmd := range s.Commands() {
		if err := d.Command(s, i, cmd); err != nil {
			return err
		}
	}
	return nil
}
