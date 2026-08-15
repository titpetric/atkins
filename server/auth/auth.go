// Package auth issues and validates the access tokens the atkins CLI
// presents to the CI/CD server.
//
// Access tokens are short-lived HS256 JWTs. They carry the session they
// were issued from, so revoking a session (logout) is enough to stop the
// refresh flow without keeping a revocation list of every access token.
package auth

import (
	"errors"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/titpetric/platform/pkg/ulid"
)

// Errors returned when a token cannot be turned into claims.
var (
	ErrEmptyToken   = errors.New("empty token")
	ErrEmptySecret  = errors.New("empty signing secret")
	ErrInvalidToken = errors.New("invalid token")
)

// JWT creates and validates signed access tokens.
type JWT struct {
	secret        string
	signingMethod *jwt.SigningMethodHMAC
}

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

// NewJWT returns a JWT signer for the given secret.
func NewJWT(secret string) *JWT {
	return &JWT{
		secret:        secret,
		signingMethod: jwt.SigningMethodHS256,
	}
}

// Create returns a signed access token for a user and session.
func (j *JWT) Create(userID, sessionID string, ttl time.Duration) (string, error) {
	if j.secret == "" {
		return "", ErrEmptySecret
	}

	claims := jwt.MapClaims{
		"user_id":    userID,
		"session_id": sessionID,
		"jti":        ulid.String(),
		"exp":        time.Now().Add(ttl).Unix(),
	}

	return jwt.NewWithClaims(j.signingMethod, claims).SignedString([]byte(j.secret))
}

// Claims validates a token and returns its claims. A leading "Bearer "
// is accepted so callers can pass the Authorization header verbatim.
func (j *JWT) Claims(token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	// Only strip "Bearer" when it is a scheme rather than the start of
	// the token itself, so "Bearerish..." is left intact.
	if rest, ok := strings.CutPrefix(token, "Bearer"); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
		token = strings.TrimSpace(rest)
	}

	if token == "" {
		return nil, ErrEmptyToken
	}
	if j.secret == "" {
		return nil, ErrEmptySecret
	}

	secret := func(*jwt.Token) (any, error) {
		return []byte(j.secret), nil
	}

	parsed, err := jwt.Parse(token, secret, jwt.WithValidMethods([]string{j.signingMethod.Alg()}))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, _ := mapClaims["user_id"].(string)
	if userID == "" {
		return nil, ErrInvalidToken
	}

	claims := &Claims{UserID: userID}
	claims.SessionID, _ = mapClaims["session_id"].(string)
	claims.JTI, _ = mapClaims["jti"].(string)
	if exp, ok := mapClaims["exp"].(float64); ok {
		claims.ExpiresAt = int64(exp)
	}

	return claims, nil
}

// UserID returns the user a token belongs to.
func (j *JWT) UserID(token string) (string, error) {
	claims, err := j.Claims(token)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
