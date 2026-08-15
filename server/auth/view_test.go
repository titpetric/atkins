package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/server/auth"
)

func TestViewToken(t *testing.T) {
	jwt := auth.NewJWT("secret")

	token := jwt.ViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	assert.Len(t, token, auth.ViewTokenLength)

	// Deriving rather than storing means the same job always yields the
	// same link, so a URL printed last week still opens.
	assert.Equal(t, token, jwt.ViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV"))

	// One job's token opens that job and nothing else.
	assert.NotEqual(t, token, jwt.ViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAW"))
	assert.True(t, jwt.ValidViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", token))
	assert.False(t, jwt.ValidViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAW", token))

	// Rotating the signing key invalidates every outstanding link,
	// which is already the documented way to revoke an instance.
	assert.False(t, auth.NewJWT("rotated").ValidViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", token))
}

func TestViewTokenRefusesEmptyInput(t *testing.T) {
	jwt := auth.NewJWT("secret")

	assert.Empty(t, jwt.ViewToken(""))
	assert.False(t, jwt.ValidViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", ""))

	// A server with no signing key mints nothing, so it must accept
	// nothing: an empty expectation would match an empty presentation.
	unsigned := auth.NewJWT("")
	assert.Empty(t, unsigned.ViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV"))
	assert.False(t, unsigned.ValidViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", ""))
	assert.False(t, unsigned.ValidViewToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "anything"))
}
