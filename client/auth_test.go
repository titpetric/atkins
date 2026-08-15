package client_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
)

func TestPrompterReadsLines(t *testing.T) {
	var out bytes.Buffer
	prompt := client.NewPrompterFrom(strings.NewReader("  ci@example.com  \nci\n"), &out)

	email, err := prompt.Line("Email", "")
	require.NoError(t, err)
	assert.Equal(t, "ci@example.com", email)

	username, err := prompt.Line("Username", "")
	require.NoError(t, err)
	assert.Equal(t, "ci", username)

	assert.Contains(t, out.String(), "Email:")
}

func TestPrompterPrefersTheEnvironment(t *testing.T) {
	t.Setenv(client.EnvEmail, "from-env@example.com")

	var out bytes.Buffer
	prompt := client.NewPrompterFrom(strings.NewReader("typed@example.com\n"), &out)

	email, err := prompt.Line("Email", client.EnvEmail)
	require.NoError(t, err)
	assert.Equal(t, "from-env@example.com", email)

	// No prompt is shown when the value came from the environment.
	assert.Empty(t, out.String())
}

func TestPrompterRefusesAPasswordWithoutATerminal(t *testing.T) {
	var out bytes.Buffer
	prompt := client.NewPrompterFrom(strings.NewReader("secret\n"), &out)

	// Reading a password from a pipe would echo it; the caller is told
	// to use the environment instead.
	_, err := prompt.Password("Password")
	assert.ErrorIs(t, err, client.ErrNoTerminal)

	t.Setenv(client.EnvPassword, "correct-horse")
	password, err := prompt.Password("Password")
	require.NoError(t, err)
	assert.Equal(t, "correct-horse", password)
}

func TestRunLogin(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, tokens("access"))

	t.Setenv(client.EnvEmail, "ci@example.com")
	t.Setenv(client.EnvPassword, "correct-horse")

	require.NoError(t, client.RunLogin(t.Context(), fake.URL))

	login := fake.requests[0]
	assert.Equal(t, "/api/user/login", login.Path)
	assert.Equal(t, "ci@example.com", login.Body["email"])
	assert.Equal(t, "correct-horse", login.Body["password"])

	stored, err := client.Open("")
	require.NoError(t, err)
	assert.Equal(t, "ci", stored.Username())
}

func TestRunLoginRequiresAServer(t *testing.T) {
	newFakeServer(t)

	assert.Error(t, client.RunLogin(t.Context(), ""))
}

func TestRunRegister(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/register", http.StatusCreated, tokens("access"))

	t.Setenv(client.EnvUsername, "ci")
	t.Setenv(client.EnvEmail, "ci@example.com")
	t.Setenv(client.EnvPassword, "correct-horse")

	require.NoError(t, client.RunRegister(t.Context(), fake.URL))

	register := fake.requests[0]
	assert.Equal(t, "/api/user/register", register.Path)
	assert.Equal(t, "ci", register.Body["username"])
	assert.Equal(t, "ci@example.com", register.Body["email"])
}

func TestRunRegisterRejectsAShortPassword(t *testing.T) {
	fake := newFakeServer(t)

	t.Setenv(client.EnvUsername, "ci")
	t.Setenv(client.EnvEmail, "ci@example.com")
	t.Setenv(client.EnvPassword, "short")

	// Caught at the prompt rather than after a round trip.
	err := client.RunRegister(t.Context(), fake.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least")
	assert.Empty(t, fake.requests)
}

func TestRunLogout(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, tokens("access"))
	fake.respond("/api/user/logout", http.StatusNoContent, nil)

	t.Setenv(client.EnvEmail, "ci@example.com")
	t.Setenv(client.EnvPassword, "correct-horse")
	require.NoError(t, client.RunLogin(t.Context(), fake.URL))

	require.NoError(t, client.RunLogout(t.Context(), ""))

	_, err := client.Open("")
	assert.ErrorIs(t, err, client.ErrNotLoggedIn)
}

func TestRunLogoutWithoutASession(t *testing.T) {
	newFakeServer(t)

	assert.ErrorIs(t, client.RunLogout(t.Context(), ""), client.ErrNotLoggedIn)
}
