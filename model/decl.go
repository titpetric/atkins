package model

// Decl represents a common variables signature {vars, env, include}.
//
// It's a base type for pipelines, jobs/tasks and steps/cmds.
type Decl struct {
	Vars    map[string]any `yaml:"vars,omitempty"`
	Include *IncludeDecl   `yaml:"include,omitempty"`
	Env     *EnvDecl       `yaml:"env,omitempty"`
}

// EnvDecl represents an environment variable declaration that can contain
// both manually-set variables and includes from external files.
type EnvDecl Decl
