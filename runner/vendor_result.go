package runner

// VendorResult reports what a vendoring run found, matched and wrote.
type VendorResult struct {
	// SourceDir is the skills directory Plan reads.
	SourceDir string

	// TargetDir is the repository directory Install writes to.
	TargetDir string

	// Found are all skills available in SourceDir.
	Found []*VendorSkill

	// Used are the skills this repository has a use for.
	Used []*VendorSkill

	// Installed are the IDs written to TargetDir. It stays empty on a
	// dry run; Pending says what a write would have done.
	Installed []string
}

// Pending returns the IDs a write would create or overwrite, leaving
// out the skills already vendored unchanged.
func (r *VendorResult) Pending() []string {
	var pending []string
	for _, skill := range r.Used {
		if skill.Status != VendorCurrent {
			pending = append(pending, skill.ID)
		}
	}
	return pending
}
