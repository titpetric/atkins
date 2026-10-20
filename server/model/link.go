package model

// ViewTokenParam is the query parameter carrying a job's view token.
// One letter, because it rides along in a URL a human copies out of a
// terminal.
const ViewTokenParam = "t"

// JobLink is where a job's page lives on the server.
//
// It lives here and outside the page package because two sides build the
// same link: the pages, for every job they point at, and the API, for
// callers that read a job and point a person somewhere to look.
// A link that differs between them is a link that works in one place and
// 403s in the other.
//
// The path is server-relative. Clients reach the server through whatever
// hostname the operator put in front of it, and inventing one here would
// be a guess; the caller already holds the address of the server it
// called.
func JobLink(jobID, viewToken string) string {
	if viewToken == "" {
		return "/job/" + jobID
	}
	return "/job/" + jobID + "?" + ViewTokenParam + "=" + viewToken
}
