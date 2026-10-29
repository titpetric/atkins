package model

import (
	"fmt"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Step is one step of a job: a task: step runs other jobs, any other step
// runs commands (run:, cmd:, cmds: or exec:). A task: value may name more
// than one job, separated by whitespace (`task: lint test build`), which run
// in sequence and match one task: step per job. An exec: argv runs without a shell; its
// display string quotes an argument holding a space or a shell character.
type Step struct {
	*Decl

	Name        string       `yaml:"name,omitempty"`
	Desc        string       `yaml:"desc,omitempty"`
	Dir         string       `yaml:"dir,omitempty"`
	Run         string       `yaml:"run,omitempty"`
	Cmd         string       `yaml:"cmd,omitempty"`
	Cmds        []string     `yaml:"cmds,omitempty"`
	Exec        []string     `yaml:"exec,omitempty"` // Program and arguments, run without a shell
	Task        string       `yaml:"task,omitempty"` // Task/job name to invoke
	If          Conditionals `yaml:"if,omitempty"`
	For         Iterators    `yaml:"for,omitempty"`
	Detach      bool         `yaml:"detach,omitempty"`
	Deferred    bool         `yaml:"deferred,omitempty"`
	Verbose     bool         `yaml:"verbose,omitempty"`
	Summarize   bool         `yaml:"summarize,omitempty"`
	Quiet       bool         `yaml:"quiet,omitempty"`
	Passthru    bool         `yaml:"passthru,omitempty"`    // If true, print output with tree indentation
	Interactive bool         `yaml:"interactive,omitempty"` // If true, stream output live and connect stdin for keyboard input
	HidePrefix  bool         `yaml:"-"`                     // If true, don't show "run:" prefix in display
}

// String returns a string representation of the step.
func (s *Step) String() string {
	switch {
	case s.Task != "":
		return "task: " + s.Task
	case s.Run != "":
		// If Run contains newlines, display <script> as the command
		if strings.Contains(s.Run, "\n") {
			return "run: <script>"
		}
		return "run: " + s.Run
	case s.Cmd != "":
		return "cmd: " + s.Cmd
	case len(s.Cmds) > 0:
		return fmt.Sprintf("cmds: <%d commands>", len(s.Cmds))
	case len(s.Exec) > 0:
		return "exec: " + s.ExecString()
	}
	return s.Name
}

// DisplayLabel returns a display label for the step, always showing prefixes.
// Execution uses it to show which kind of operation runs.
func (s *Step) DisplayLabel() string {
	if s.Desc != "" {
		return s.Desc
	}
	switch {
	case s.Task != "":
		return "task: " + s.Task
	case s.Run != "":
		// If Run contains newlines, display <script> as the command
		if strings.Contains(s.Run, "\n") {
			return "run: <script>"
		}
		return "run: " + s.Run
	case s.Cmd != "":
		return "cmd: " + s.Cmd
	case len(s.Cmds) > 0:
		return fmt.Sprintf("cmds: <%d commands>", len(s.Cmds))
	case len(s.Exec) > 0:
		return "exec: " + s.ExecString()
	}
	return s.Name
}

// Label returns a structured label for the step with information about how to display it.
// showPrefix controls whether to include the type prefix (e.g., "run:", "task:").
// HidePrefix hides the prefix in every case (for simple shorthand tasks).
func (s *Step) Label(showPrefix bool) *Label {
	switch {
	case s.Task != "":
		return &Label{
			Text:       s.Task,
			Type:       "task",
			ShowPrefix: showPrefix && !s.HidePrefix,
		}
	case s.Run != "":
		text := s.Run
		if strings.Contains(text, "\n") {
			text = "<script>"
		}
		return &Label{
			Text:       text,
			Type:       "run",
			ShowPrefix: showPrefix && !s.HidePrefix,
		}
	case s.Cmd != "":
		return &Label{
			Text:       s.Cmd,
			Type:       "cmd",
			ShowPrefix: showPrefix && !s.HidePrefix,
		}
	case len(s.Cmds) > 0:
		return &Label{
			Text:       fmt.Sprintf("<%d commands>", len(s.Cmds)),
			Type:       "cmds",
			ShowPrefix: showPrefix && !s.HidePrefix,
		}
	case len(s.Exec) > 0:
		return &Label{
			Text:       s.ExecString(),
			Type:       "exec",
			ShowPrefix: showPrefix && !s.HidePrefix,
		}
	}
	return &Label{
		Text:       s.Name,
		Type:       "default",
		ShowPrefix: false,
	}
}

// Tasks returns the job names of the task: field, none for a command step.
func (s *Step) Tasks() []string {
	return strings.Fields(s.Task)
}

// Commands returns all executable commands from this step as a slice.
// For steps with a single command (Run/Cmd), returns a slice with one element.
// For steps with a command list (Cmds), returns the full slice.
// An exec: step returns its argv rendered as one display string.
// Returns an empty slice for Task steps.
func (s *Step) Commands() []string {
	if len(s.Cmds) > 0 {
		return s.Cmds
	}
	if s.Run != "" {
		return []string{s.Run}
	}
	if s.Cmd != "" {
		return []string{s.Cmd}
	}
	if len(s.Exec) > 0 {
		return []string{s.ExecString()}
	}
	return []string{}
}

// ExecString returns the exec: argv as one display string.
func (s *Step) ExecString() string {
	parts := make([]string, 0, len(s.Exec))
	for _, arg := range s.Exec {
		if arg == "" || strings.ContainsAny(arg, " \t\n'\"\\$`;&|<>()*?[]{}~#") {
			arg = strconv.Quote(arg)
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

// IsDeferred returns true if the step sets deferred.
func (s *Step) IsDeferred() bool {
	return s.Deferred
}

// UnmarshalYAML implements custom unmarshalling for Step to accept the shorthand formats and decode Decl.
func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		// Simple string step - treat as a Run command
		s.Run = strings.TrimSpace(node.Value)
		s.Name = strings.TrimSpace(node.Value)
		return nil
	}

	if node.Kind == yaml.MappingNode {
		type rawStep Step
		if err := node.Decode((*rawStep)(s)); err != nil {
			return err
		}

		var ds DeferredStep
		if err := node.Decode(&ds); err != nil {
			return err
		}

		if ds.Defer != nil {
			if s.Run != "" {
				return fmt.Errorf("step has run %s and defer %s: use {defer} or {run, deferred=true}", s.Run, ds.Defer.Run)
			}

			*s = *ds.Defer
			s.Deferred = true

			return nil
		}

		// Ensure Decl exists and vars/include decode properly
		if err := ensureDeclInitialized(node, &s.Decl); err != nil {
			return err
		}

		// Trim spaces from Run, Cmd, and Cmds after decoding
		s.Run = strings.TrimSpace(s.Run)
		s.Cmd = strings.TrimSpace(s.Cmd)
		for i, cmd := range s.Cmds {
			s.Cmds[i] = strings.TrimSpace(cmd)
		}

		// An exec: step prints its output like a job's cmd: shorthand
		// does, unless the step sets passthru: false.
		if len(s.Exec) > 0 && !hasKey(node, "passthru") {
			s.Passthru = true
		}

		return nil
	}

	return fmt.Errorf("invalid step format: expected string or object, got %v", node.Kind)
}

// hasKey reports whether a mapping node sets the key.
func hasKey(node *yaml.Node, key string) bool {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return true
		}
	}
	return false
}

// Dispatch returns d.Task for a task: step.
// Dispatch returns the first error of d.Command, called once per command.
// Dispatch returns nil once every command has run.
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
