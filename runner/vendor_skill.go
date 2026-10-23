package runner

import (
	"github.com/titpetric/atkins/model"
)

// VendorStatus is what installing a skill would do to the repository.
type VendorStatus string

// The states Plan sets on a used skill.
const (
	// VendorNew means the repository doesn't carry the skill yet.
	VendorNew VendorStatus = "new"

	// VendorCurrent means an identical copy is already vendored.
	VendorCurrent VendorStatus = "up to date"

	// VendorChanged means the vendored copy differs from the source and
	// would be overwritten.
	VendorChanged VendorStatus = "changed"
)

// VendorSkill is a candidate skill file, with the reason Plan picks it
// for vendoring once it matches against a repository.
type VendorSkill struct {
	// ID is the skill namespace, taken from the file name.
	ID string

	// Path is the absolute path of the source skill file.
	Path string

	// DocPath is the markdown companion beside Path, empty when the
	// skill carries none. Install copies it alongside the skill file.
	DocPath string

	// Pipeline is the parsed skill.
	Pipeline *model.Pipeline

	// Reason records why the repository uses the skill, e.g. the path that
	// satisfied its when: block, or the pipeline that references it.
	Reason string

	// Status is what installing this skill would do, set for used skills.
	Status VendorStatus

	// Added and Removed are the line counts installing this skill would
	// add to and remove from the vendored copy.
	Added   int
	Removed int

	// Diff is the unified diff from the vendored copy to the source,
	// empty when there is nothing to change.
	Diff string
}
