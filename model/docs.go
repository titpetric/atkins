// Package model holds the types a pipeline file decodes into: Pipeline, Job,
// Step and the declarations they share (Decl: vars, env and include). The
// types accept both the GitHub Actions and the Taskfile spellings of a field,
// and their UnmarshalYAML methods take the shorthand forms, such as a job or
// a step written as one command string.
//
// The package describes a pipeline and runs nothing. Walk and Dispatch visit
// jobs and steps in a fixed order for the runner and the linter, which
// decide what a visit does. types.mig.go holds the models of the event
// database, which mig generates from schema/*.up.sql.
package model
