package auth_test

import (
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/server/auth"
)

func TestCreateAndParse(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token, err := jwt.Create("user-1", "session-1", time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwt.Claims(token)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "session-1", claims.SessionID)
	assert.NotEmpty(t, claims.JTI)
	assert.Positive(t, claims.ExpiresAt)
}

func TestClaimsAcceptsBearerPrefix(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token, err := jwt.Create("user-1", "session-1", time.Hour)
	assert.NoError(t, err)

	userID, err := jwt.UserID("Bearer " + token)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestClaimsRejectsForeignSignature(t *testing.T) {
	token, err := auth.NewJWT("secret").Create("user-1", "session-1", time.Hour)
	assert.NoError(t, err)

	_, err = auth.NewJWT("other-secret").Claims(token)
	assert.Error(t, err)
}

func TestClaimsRejectsExpiredToken(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token, err := jwt.Create("user-1", "session-1", -time.Minute)
	assert.NoError(t, err)

	_, err = jwt.Claims(token)
	assert.Error(t, err)
}

func TestClaimsRejectsEmptyInput(t *testing.T) {
	jwt := auth.NewJWT("secret")

	_, err := jwt.Claims("")
	assert.ErrorIs(t, err, auth.ErrEmptyToken)

	_, err = jwt.Claims("Bearer ")
	assert.ErrorIs(t, err, auth.ErrEmptyToken)
}

func TestEmptySecretIsRefused(t *testing.T) {
	jwt := auth.NewJWT("")

	_, err := jwt.Create("user-1", "session-1", time.Hour)
	assert.ErrorIs(t, err, auth.ErrEmptySecret)

	_, err = jwt.Claims("anything")
	assert.ErrorIs(t, err, auth.ErrEmptySecret)
}
