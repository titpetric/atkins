package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

// ViewTokenLength is how many base64url characters of the digest a job
// link carries: 22 characters is 132 bits, which is not guessable and
// still fits on one terminal line next to a ULID.
const ViewTokenLength = 22

// viewTokenDomain separates view tokens from anything else the signing
// key is ever used for, so a token minted here can never be mistaken
// for a token minted elsewhere under the same secret.
const viewTokenDomain = "atkins:job-view:"

// ViewToken returns the unguessable half of a job page URL.
//
// It is derived rather than stored: an HMAC of the job ID under the
// server's signing key. There is no column to migrate, nothing extra to
// leak from a database dump, and rotating the signing key invalidates
// every outstanding link at once — which is already the documented way
// to revoke access to an instance.
func (j *JWT) ViewToken(jobID string) string {
	if j.secret == "" || jobID == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(j.secret))
	mac.Write([]byte(viewTokenDomain + jobID))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:ViewTokenLength]
}

// ValidViewToken reports whether presented is the view token for jobID.
//
// A server with no signing key validates nothing: it cannot mint a
// token either, so accepting one would be accepting anything.
func (j *JWT) ValidViewToken(jobID, presented string) bool {
	expected := j.ViewToken(jobID)
	if expected == "" || presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) == 1
}
