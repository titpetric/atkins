# Package ./server/auth

```go
import (
	"github.com/titpetric/atkins/server/auth"
}
```

Package auth issues and validates the access tokens the atkins CLI
presents to the CI/CD server.

Access tokens are short-lived HS256 JWTs. They carry the session they
were issued from, so revoking a session (logout) is enough to stop the
refresh flow without keeping a revocation list of every access token.

## Types

<details>
<summary><code>type Claims</code></summary>

```go
// Claims is the decoded payload of an access token.
type Claims struct {
	// UserID is the authenticated user.
	UserID string `json:"user_id"`

	// SessionID ties the token to the login it came from.
	SessionID string `json:"session_id"`

	// JTI identifies this token.
	JTI string `json:"jti"`

	// ExpiresAt is the unix timestamp from the `exp` claim.
	ExpiresAt int64 `json:"exp"`
}
```

</details>

<details>
<summary><code>type JWT</code></summary>

```go
// JWT creates and validates signed access tokens.
type JWT struct {
	secret        string
	signingMethod *jwt.SigningMethodHMAC
}
```

</details>

## Consts

<details>
<summary><code>const ViewTokenLength</code></summary>

```go
// ViewTokenLength is how many base64url characters of the digest a job
// link carries: 22 characters is 132 bits, which is not guessable and
// still fits on one terminal line next to a ULID.
const ViewTokenLength = 22
```

</details>

## Vars

<details>
<summary><code>var ErrEmptyToken, ErrEmptySecret, ErrInvalidToken</code></summary>

```go
// Errors returned when a token cannot be turned into claims.
var (
	ErrEmptyToken   = errors.New("empty token")
	ErrEmptySecret  = errors.New("empty signing secret")
	ErrInvalidToken = errors.New("invalid token")
)
```

</details>

## Function symbols

- `func NewJWT (secret string) *JWT`
- `func (*JWT) Claims (token string) (*Claims, error)`
- `func (*JWT) Create (userID,sessionID string, ttl time.Duration) (string, error)`
- `func (*JWT) UserID (token string) (string, error)`
- `func (*JWT) ValidViewToken (jobID,presented string) bool`
- `func (*JWT) ViewToken (jobID string) string`

### NewJWT

NewJWT returns a JWT signer for the given secret.

```go
func NewJWT(secret string) *JWT
```

### Claims

Claims validates a token and returns its claims. A leading "Bearer "
is accepted so callers can pass the Authorization header verbatim.

```go
func (*JWT) Claims(token string) (*Claims, error)
```

### Create

Create returns a signed access token for a user and session.

```go
func (*JWT) Create(userID, sessionID string, ttl time.Duration) (string, error)
```

### UserID

UserID returns the user a token belongs to.

```go
func (*JWT) UserID(token string) (string, error)
```

### ValidViewToken

ValidViewToken reports whether presented is the view token for jobID.

A server with no signing key validates nothing: it cannot mint a
token either, so accepting one would be accepting anything.

```go
func (*JWT) ValidViewToken(jobID, presented string) bool
```

### ViewToken

ViewToken returns the unguessable half of a job page URL.

It is derived rather than stored: an HMAC of the job ID under the
server's signing key. There is no column to migrate, nothing extra to
leak from a database dump, and rotating the signing key invalidates
every outstanding link at once — which is already the documented way
to revoke access to an instance.

```go
func (*JWT) ViewToken(jobID string) string
```
