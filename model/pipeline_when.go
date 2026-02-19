package model

// PipelineWhen is a list of files that need to exist somewhere to
// enable the pipeline, e.g. compose.yml for compose pipeline.
type PipelineWhen struct {
	Files []string `yaml:"files"`
}
