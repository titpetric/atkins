// Package runner loads atkins pipelines and runs their jobs.
//
// A run starts from a pipeline file: DiscoverConfig finds it by walking up
// from the working directory, LoadPipeline parses it, and a SkillsLoader
// adds the skill pipelines from the .atkins/skills folders that apply. A
// skill applies when its when: block matches a file near the working
// directory, or when it has no when: block.
//
// RunPipeline runs the requested jobs. Before any job starts, the pipeline
// resolves the jobs and their dependencies into a run order and builds a
// tree node for every job a step can invoke. The first render of the tree
// shows the whole run. An Executor then runs each job's steps, with
// their for: loops, task: references and conditions, and the tree and the
// event log record what each step did.
//
// Values reach a command through interpolation: `${{ expression }}` reads
// variables and expressions, and `$(command)` substitutes the output of a
// shell command. ContextVariables holds the variables of a run and can
// resolve a pending one on its first read.
//
// A project cloned inside a workspace checkout sees the workspace's skills
// through SkillsLoader.AddParentWorkspaces. Inside home that walk stops
// below home: $HOME/.atkins/skills loads on its own as the global skills,
// and the folders above home are not the user's to configure. Outside home
// the walk goes on to the filesystem root, the same as DiscoverConfig.
//
// A task reference resolves through a TaskResolver: a skill's own jobs
// first, then every loaded pipeline, with a leading colon skipping the
// skill. The Vendorer copies the skills a repository uses into the
// repository's own .atkins/skills.
package runner
