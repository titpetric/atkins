package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/agent"
)

func TestAliasStore_New(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	store := agent.NewAliasStore()
	assert.NotNil(t, store)
}

func TestAliasStore_Add_New(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	store := agent.NewAliasStore()
	store.Add("deploy", "docker:push")

	result := store.Match("deploy")
	assert.Equal(t, "docker:push", result)
}

func TestAliasStore_Add_Update(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	store := agent.NewAliasStore()
	store.Add("deploy", "docker:push")
	store.Add("deploy", "docker:build") // Update

	result := store.Match("deploy")
	assert.Equal(t, "docker:build", result)
}

func TestAliasStore_Match_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	store := agent.NewAliasStore()
	store.Add("Server Name", "uname -n")

	assert.Equal(t, "uname -n", store.Match("server name"))
	assert.Equal(t, "uname -n", store.Match("SERVER NAME"))
	assert.Equal(t, "uname -n", store.Match("Server Name"))
}

func TestAliasStore_Match_WithPunctuation(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	store := agent.NewAliasStore()
	store.Add("build it", "go:build")

	assert.Equal(t, "go:build", store.Match("build it!"))
	assert.Equal(t, "go:build", store.Match("build it?"))
	assert.Equal(t, "go:build", store.Match("build it."))
}

func TestAliasStore_Match_WithFillerWords(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	store := agent.NewAliasStore()
	store.Add("server name", "uname -n")

	// Should match with filler words stripped
	assert.Equal(t, "uname -n", store.Match("give me the server name"))
	assert.Equal(t, "uname -n", store.Match("show my server name"))
}

func TestAliasStore_Match_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	store := agent.NewAliasStore()
	store.Add("deploy", "docker:push")

	assert.Empty(t, store.Match("unknown"))
	assert.Empty(t, store.Match("deployer")) // Not an exact match
}

func TestParseCorrection_IfISay(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("if i say deploy, run docker:push")
	assert.True(t, ok)
	assert.Equal(t, "deploy", phrase)
	assert.Equal(t, "docker:push", task)
}

func TestParseCorrection_IfISayToRun(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("if i say to run tests, run go:test")
	assert.True(t, ok)
	assert.Equal(t, "tests", phrase)
	assert.Equal(t, "go:test", task)
}

func TestParseCorrection_WhenISay(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("when i say build, run make build")
	assert.True(t, ok)
	assert.Equal(t, "build", phrase)
	assert.Equal(t, "make build", task)
}

func TestParseCorrection_Map(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("map test to go:test")
	assert.True(t, ok)
	assert.Equal(t, "test", phrase)
	assert.Equal(t, "go:test", task)
}

func TestParseCorrection_Alias(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("alias server name to uname -n")
	assert.True(t, ok)
	assert.Equal(t, "server name", phrase)
	assert.Equal(t, "uname -n", task)
}

func TestParseCorrection_ShouldRun(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("run tests should run go:test")
	assert.True(t, ok)
	assert.Equal(t, "run tests", phrase)
	assert.Equal(t, "go:test", task)
}

func TestParseCorrection_Means(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection("deploy means docker:push")
	assert.True(t, ok)
	assert.Equal(t, "deploy", phrase)
	assert.Equal(t, "docker:push", task)
}

func TestParseCorrection_Invalid(t *testing.T) {
	tests := []string{
		"just some text",
		"",
		"hello world",
		"if i say", // Incomplete
		"map to",   // Missing phrase
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, _, ok := agent.ParseCorrection(input)
			assert.False(t, ok)
		})
	}
}

func TestParseCorrection_WithQuotes(t *testing.T) {
	phrase, task, ok := agent.ParseCorrection(`alias "run tests" to "go:test"`)
	assert.True(t, ok)
	assert.Equal(t, "run tests", phrase)
	assert.Equal(t, "go:test", task)
}
