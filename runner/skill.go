package runner

import "github.com/titpetric/atkins/model"

// Skill is one skill file found on disk.
//
// A skill is a pipeline file whose name is the job namespace: go.yml
// contributes go:build, go:test and the rest. Beside it a skill may
// carry a markdown companion of the same base name, go.md, which is a
// usage guide rather than a job list; `atkins --help` prints it under
// the skill.
type Skill struct {
	// Pipeline is the parsed skill. Its ID is the file name without the
	// extension, and prefixes every job the skill declares.
	Pipeline *model.Pipeline

	// Path is the absolute path of the skill file.
	Path string

	// DocPath is the markdown companion beside the skill file, empty
	// when the skill carries none.
	DocPath string

	// Doc is the contents of DocPath, with surrounding blank lines
	// trimmed.
	Doc string

	// Dir is the directory the skill runs in: the folder that satisfied
	// its when: block, or the workspace root when it has none. It is
	// only resolved for an active skill.
	Dir string

	// Active reports whether the skill's when: block is satisfied from
	// the directory the scan started in. Only an active skill
	// contributes its jobs to a run.
	Active bool
}

// ID returns the skill namespace, the file name without its extension.
func (s *Skill) ID() string {
	if s == nil || s.Pipeline == nil {
		return ""
	}
	return s.Pipeline.ID
}
