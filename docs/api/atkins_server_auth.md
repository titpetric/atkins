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

```go
// JWT creates and validates signed access tokens.
type JWT struct {
	secret        string
	signingMethod *jwt.SigningMethodHMAC
}
```

## Vars

```go
// Errors returned when a token cannot be turned into claims.
var (
	ErrEmptyToken   = errors.New("empty token")
	ErrEmptySecret  = errors.New("empty signing secret")
	ErrInvalidToken = errors.New("invalid token")
)
```

## Function symbols

- `func NewJWT (secret string) *JWT`
- `func (*JWT) Claims (token string) (*Claims, error)`
- `func (*JWT) Create (userID,sessionID string, ttl time.Duration) (string, error)`
- `func (*JWT) UserID (token string) (string, error)`

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
