package model

// JobVisibility decides who may read a job and the output it captured.
type JobVisibility = string

// Job visibility values.
const (
	// VisibilityPrivate is the default. Over the API a job is readable
	// by the user whose dispatch created it (and by anything under that
	// job's tree); admins and agents still see everything. The job page
	// requires the per-job token atkins prints as part of the URL.
	VisibilityPrivate JobVisibility = "private"

	// VisibilityPublic is the single-team instance the CI/CD server
	// first targeted: every authenticated user reads every job, and the
	// job page opens for anyone holding the URL. An admin picks it; a
	// new instance starts private.
	VisibilityPublic JobVisibility = "public"
)

// ValidJobVisibility reports whether visibility is a known value.
func ValidJobVisibility(visibility JobVisibility) bool {
	return visibility == VisibilityPrivate || visibility == VisibilityPublic
}
