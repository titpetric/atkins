package model

// PipelineWhen is a list of files that need to exist somewhere to
// enable the pipeline, e.g. compose.yml for compose pipeline.
//
// A job carries the same block to gate one job of a pipeline: where
// its when: block does not match, the job stays out of the listing and
// running it fails with an error.
type PipelineWhen struct {
	// Files lists the patterns, any of which activates the block. Each
	// pattern resolves in the starting folder and then in every parent
	// folder, nearest first.
	Files []string `yaml:"files" json:"files"`

	// Here restricts the search to the starting folder, so a match in
	// a parent folder does not count.
	Here bool `yaml:"here,omitempty" json:"here,omitempty"`

	// Contains is a regular expression the matched file must match.
	// With Contains, a pattern resolving to a folder never matches.
	Contains string `yaml:"contains,omitempty" json:"contains,omitempty"`
}

// IsEmpty reports whether the block lists no files, leaving the pipeline or job active everywhere.
func (w *PipelineWhen) IsEmpty() bool {
	return w == nil || len(w.Files) == 0
}
