package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/config"
)

func TestDefaultIsComplete(t *testing.T) {
	// The embedded document is the source of every default, so a field
	// missing from it would silently default to a zero value.
	cfg, err := config.Default()
	require.NoError(t, err)

	assert.Equal(t, config.Version, cfg.Version)
	assert.True(t, cfg.Client.Dispatch)
	assert.Positive(t, cfg.Client.Timeout)

	assert.NotEmpty(t, cfg.Server.Addr)
	assert.NotEmpty(t, cfg.Server.Database)
	assert.NotEmpty(t, cfg.Server.Connection)
	assert.Positive(t, cfg.Server.TokenTTL)
	assert.Positive(t, cfg.Server.SessionTTL)
	assert.Positive(t, cfg.Server.MaxJobDepth)
	assert.Positive(t, cfg.Server.LeaseTTL)

	assert.NotEmpty(t, cfg.Agent.DataDir)
	assert.NotEmpty(t, cfg.Agent.Shell)
	assert.Positive(t, cfg.Agent.PollInterval)
	assert.Positive(t, cfg.Agent.JobTimeout)
	assert.Positive(t, cfg.Agent.HeartbeatInterval)

	// A signing key must never have a default.
	assert.Empty(t, cfg.Server.SigningKey)
	assert.Empty(t, cfg.Server.AgentToken)
}

func TestDefaultValidates(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)
	assert.NoError(t, cfg.Validate())
}

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	cfg, err := config.Default()
	require.NoError(t, err)

	cfg.Client.Server = "https://ci.example.com"
	cfg.Agent.Labels = []string{"linux", "arm64"}
	require.NoError(t, cfg.Save(path))

	// The document holds secrets; it must not be world readable.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	reloaded, err := config.LoadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "https://ci.example.com", reloaded.Client.Server)
	assert.Equal(t, []string{"linux", "arm64"}, reloaded.Agent.Labels)
}

func TestPartialDocumentKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".atkins", "config.yml")

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("client:\n  server: https://ci.example.com\n"), 0o600))

	cfg, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "https://ci.example.com", cfg.Client.Server)
	// Untouched fields keep the embedded defaults rather than becoming
	// zero values.
	assert.Equal(t, "/bin/sh", cfg.Agent.Shell)
	assert.Equal(t, int64(3), cfg.Server.MaxJobDepth)
}

func TestEnvironmentOverlay(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	cfg.Client.Server = "https://configured.example.com"

	cfg.ApplyEnvironment([]string{
		"ATKINS_SERVER=https://from-env.example.com",
		"ATKINS_LEASE_TTL=42m",
		"ATKINS_MAX_JOB_DEPTH=7",
		"ATKINS_ALLOW_REGISTRATION=true",
		"ATKINS_LABELS=linux, arm64 ,,docker",
	})

	assert.Equal(t, "https://from-env.example.com", cfg.Client.Server)
	assert.Equal(t, 42*time.Minute, cfg.Server.LeaseTTL)
	assert.Equal(t, int64(7), cfg.Server.MaxJobDepth)
	assert.True(t, cfg.Server.AllowRegistration)
	assert.Equal(t, []string{"linux", "arm64", "docker"}, cfg.Client.Labels)
}

func TestEmptyEnvironmentLeavesConfigAlone(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	cfg.Client.Server = "https://configured.example.com"
	cfg.Server.LeaseTTL = 9 * time.Minute

	// An exported-but-empty variable means "not configured here", not
	// "clear the configured value".
	cfg.ApplyEnvironment([]string{"ATKINS_SERVER=", "ATKINS_LEASE_TTL="})

	assert.Equal(t, "https://configured.example.com", cfg.Client.Server)
	assert.Equal(t, 9*time.Minute, cfg.Server.LeaseTTL)
}

func TestUnparseableEnvironmentIsIgnored(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	before := cfg.Server.LeaseTTL
	cfg.ApplyEnvironment([]string{"ATKINS_LEASE_TTL=not-a-duration"})

	assert.Equal(t, before, cfg.Server.LeaseTTL)
}

func TestValidateFillsEmptyValues(t *testing.T) {
	cfg := &config.Config{}
	require.NoError(t, cfg.Validate())

	assert.Equal(t, config.Version, cfg.Version)
	assert.NotEmpty(t, cfg.Server.Addr)
	assert.Positive(t, cfg.Agent.PollInterval)
	assert.Equal(t, "/bin/sh", cfg.Agent.Shell)
}

func TestValidateRejectsBadValues(t *testing.T) {
	tests := map[string]func(*config.Config){
		"server url without a scheme": func(c *config.Config) {
			c.Client.Server = "ci.example.com"
		},
		"addr without a port": func(c *config.Config) {
			c.Server.Addr = "localhost"
		},
		"database without a dsn scheme": func(c *config.Config) {
			c.Server.Database = "atkins.db"
		},
		"session shorter than a token": func(c *config.Config) {
			c.Server.TokenTTL = 2 * time.Hour
			c.Server.SessionTTL = time.Hour
		},
		"heartbeat longer than the job timeout": func(c *config.Config) {
			c.Agent.HeartbeatInterval = time.Hour
			c.Agent.JobTimeout = time.Minute
		},
		"version from the future": func(c *config.Config) {
			c.Version = config.Version + 1
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.Default()
			require.NoError(t, err)

			mutate(cfg)
			assert.Error(t, cfg.Validate())
		})
	}
}

func TestDiscoverWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "server", "api")

	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".atkins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".atkins", "config.yml"), []byte("version: 1\n"), 0o600))

	found := config.Discover(nested)
	require.NotEmpty(t, found)
	assert.Equal(t, filepath.Join(root, ".atkins", "config.yml"), found)
}

func TestDiscoverWithoutDocument(t *testing.T) {
	assert.Empty(t, config.Discover(t.TempDir()))
}

func TestProjectPathForNewDocument(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, filepath.Join(dir, ".atkins", "config.yml"), config.ProjectPath(dir))
}

func TestFields(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	fields := cfg.Fields()
	require.NotEmpty(t, fields)

	paths := make([]string, 0, len(fields))
	for _, field := range fields {
		paths = append(paths, field.Path)
	}

	assert.Contains(t, paths, "client.server")
	assert.Contains(t, paths, "server.signing_key")
	assert.Contains(t, paths, "agent.data_dir")
	// The document version is derived, not edited.
	assert.NotContains(t, paths, "version")
}

func TestFieldSet(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	lease, ok := cfg.Field("server.lease_ttl")
	require.True(t, ok)
	require.NoError(t, lease.Set("42m"))
	assert.Equal(t, 42*time.Minute, cfg.Server.LeaseTTL)
	assert.Error(t, lease.Set("not-a-duration"))

	labels, ok := cfg.Field("agent.labels")
	require.True(t, ok)
	require.NoError(t, labels.Set(" linux , arm64 "))
	assert.Equal(t, []string{"linux", "arm64"}, cfg.Agent.Labels)

	dispatch, ok := cfg.Field("client.dispatch")
	require.True(t, ok)
	require.NoError(t, dispatch.Set("false"))
	assert.False(t, cfg.Client.Dispatch)
	assert.Error(t, dispatch.Set("maybe"))

	depth, ok := cfg.Field("server.max_job_depth")
	require.True(t, ok)
	require.NoError(t, depth.Set("5"))
	assert.Equal(t, int64(5), cfg.Server.MaxJobDepth)
	assert.Error(t, depth.Set("many"))
}

func TestFieldSecretsAreMasked(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	cfg.Server.SigningKey = "super-secret-value"

	field, ok := cfg.Field("server.signing_key")
	require.True(t, ok)
	require.True(t, field.Secret)

	display := field.Display()
	assert.NotContains(t, display, "secret")
	assert.Contains(t, display, "*")
	assert.Equal(t, "super-secret-value", field.String())
}

func TestFieldDisplayMarksEmpty(t *testing.T) {
	cfg, err := config.Default()
	require.NoError(t, err)

	field, ok := cfg.Field("client.server")
	require.True(t, ok)
	assert.Equal(t, "(not set)", field.Display())
}

func TestEnvironmentNames(t *testing.T) {
	names := config.EnvironmentNames()

	assert.Equal(t, "ATKINS_SERVER", names["client.server"])
	assert.Equal(t, "ATKINS_SIGNING_KEY", names["server.signing_key"])
	assert.Equal(t, "ATKINS_DATA_DIR", names["agent.data_dir"])
}

func TestSplitList(t *testing.T) {
	assert.Nil(t, config.SplitList(""))
	assert.Nil(t, config.SplitList(" , , "))
	assert.Equal(t, []string{"a", "b"}, config.SplitList(" a , b "))
}
