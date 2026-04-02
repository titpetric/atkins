package model

// GitStats holds +/- line counts from git diff.
type GitStats struct {
	Added   int
	Removed int
}
