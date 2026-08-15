package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/auth"
)

func TestCreateAndParse(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token, err := jwt.Create("user-1", "session-1", time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := jwt.Claims(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "session-1", claims.SessionID)
	assert.NotEmpty(t, claims.JTI)
	assert.Positive(t, claims.ExpiresAt)
}

func TestClaimsAcceptsBearerPrefix(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token, err := jwt.Create("user-1", "session-1", time.Hour)
	require.NoError(t, err)

	userID, err := jwt.UserID("Bearer " + token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestClaimsRejectsForeignSignature(t *testing.T) {
	token, err := auth.NewJWT("secret").Create("user-1", "session-1", time.Hour)
	require.NoError(t, err)

	_, err = auth.NewJWT("other-secret").Claims(token)
	assert.Error(t, err)
}

func TestClaimsRejectsExpiredToken(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token, err := jwt.Create("user-1", "session-1", -time.Minute)
	require.NoError(t, err)

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
