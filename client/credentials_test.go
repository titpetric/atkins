package client_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
)

// credentialsFile points the store at a temporary file.
func credentialsFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "credentials.json")
	t.Setenv("ATKINS_CREDENTIALS", path)

	return path
}

func TestLoadStoreWithoutFile(t *testing.T) {
	credentialsFile(t)

	store, err := client.LoadStore()
	require.NoError(t, err)
	assert.Empty(t, store.Default)
	assert.Empty(t, store.Servers)

	_, ok := store.Get("")
	assert.False(t, ok)
}

func TestStoreRoundTrip(t *testing.T) {
	path := credentialsFile(t)

	store, err := client.LoadStore()
	require.NoError(t, err)

	store.Set(&client.Credential{
		Server:       "https://ci.example.com",
		Username:     "ci",
		Token:        "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, store.Save())

	// Tokens are secrets; the file must not be world readable.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	reloaded, err := client.LoadStore()
	require.NoError(t, err)
	assert.Equal(t, "https://ci.example.com", reloaded.Default)

	credential, ok := reloaded.Get("")
	require.True(t, ok)
	assert.Equal(t, "ci", credential.Username)
	assert.Equal(t, "access", credential.Token)

	// A trailing slash names the same server.
	_, ok = reloaded.Get("https://ci.example.com/")
	assert.True(t, ok)
}

func TestStoreRemovePicksAnotherDefault(t *testing.T) {
	credentialsFile(t)

	store, err := client.LoadStore()
	require.NoError(t, err)

	store.Set(&client.Credential{Server: "https://one.example.com"})
	store.Set(&client.Credential{Server: "https://two.example.com"})
	require.Equal(t, "https://two.example.com", store.Default)

	store.Remove("https://two.example.com")
	assert.Equal(t, "https://one.example.com", store.Default)

	store.Remove("https://one.example.com")
	assert.Empty(t, store.Default)
}

func TestCredentialExpired(t *testing.T) {
	var missing *client.Credential
	assert.True(t, missing.Expired())

	assert.True(t, (&client.Credential{}).Expired())
	assert.True(t, (&client.Credential{ExpiresAt: time.Now().Add(-time.Hour).Unix()}).Expired())

	// Within the refresh skew, a token counts as expired.
	assert.True(t, (&client.Credential{ExpiresAt: time.Now().Add(10 * time.Second).Unix()}).Expired())
	assert.False(t, (&client.Credential{ExpiresAt: time.Now().Add(time.Hour).Unix()}).Expired())
}

func TestOpenWithoutCredentials(t *testing.T) {
	credentialsFile(t)

	_, err := client.Open("")
	assert.ErrorIs(t, err, client.ErrNotLoggedIn)
}

func TestNormalizeServer(t *testing.T) {
	assert.Equal(t, "https://ci.example.com", client.NormalizeServer(" https://ci.example.com/ "))
	assert.Equal(t, "", client.NormalizeServer("  "))
}
