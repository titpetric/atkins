package greeting_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/agent"
)

func TestGreeter_NewGreeter(t *testing.T) {
	g := agent.NewGreeter()
	assert.NotNil(t, g)
}

func TestGreeter_Match_English(t *testing.T) {
	g := agent.NewGreeter()

	greetings := []string{"hi", "hey", "hello", "howdy", "yo", "sup", "heya"}
	for _, greeting := range greetings {
		t.Run(greeting, func(t *testing.T) {
			response := g.Match(greeting)
			assert.NotEmpty(t, response, "expected response for %q", greeting)
		})
	}
}

func TestGreeter_Match_Thanks(t *testing.T) {
	g := agent.NewGreeter()

	thanks := []string{"thanks", "thank you", "thx", "ty", "cheers"}
	for _, thank := range thanks {
		t.Run(thank, func(t *testing.T) {
			response := g.Match(thank)
			assert.NotEmpty(t, response, "expected response for %q", thank)
		})
	}
}

func TestGreeter_Match_Spanish(t *testing.T) {
	g := agent.NewGreeter()

	spanish := []string{"hola", "buenas", "gracias"}
	for _, word := range spanish {
		t.Run(word, func(t *testing.T) {
			response := g.Match(word)
			assert.NotEmpty(t, response, "expected response for %q", word)
		})
	}
}

func TestGreeter_Match_French(t *testing.T) {
	g := agent.NewGreeter()

	french := []string{"bonjour", "salut", "merci"}
	for _, word := range french {
		t.Run(word, func(t *testing.T) {
			response := g.Match(word)
			assert.NotEmpty(t, response, "expected response for %q", word)
		})
	}
}

func TestGreeter_Match_German(t *testing.T) {
	g := agent.NewGreeter()

	german := []string{"hallo", "moin", "danke"}
	for _, word := range german {
		t.Run(word, func(t *testing.T) {
			response := g.Match(word)
			assert.NotEmpty(t, response, "expected response for %q", word)
		})
	}
}

func TestGreeter_Match_Italian(t *testing.T) {
	g := agent.NewGreeter()

	italian := []string{"ciao", "salve", "grazie"}
	for _, word := range italian {
		t.Run(word, func(t *testing.T) {
			response := g.Match(word)
			assert.NotEmpty(t, response, "expected response for %q", word)
		})
	}
}

func TestGreeter_Match_Portuguese(t *testing.T) {
	g := agent.NewGreeter()

	portuguese := []string{"oi", "obrigado", "valeu"}
	for _, word := range portuguese {
		t.Run(word, func(t *testing.T) {
			response := g.Match(word)
			assert.NotEmpty(t, response, "expected response for %q", word)
		})
	}
}

func TestGreeter_Match_WithPunctuation(t *testing.T) {
	g := agent.NewGreeter()

	assert.NotEmpty(t, g.Match("hi!"))
	assert.NotEmpty(t, g.Match("hello?"))
	assert.NotEmpty(t, g.Match("hey."))
	assert.NotEmpty(t, g.Match("thanks!!!"))
}

func TestGreeter_Match_WithPrefix(t *testing.T) {
	g := agent.NewGreeter()

	assert.NotEmpty(t, g.Match("hi there"))
	assert.NotEmpty(t, g.Match("hello world"))
	assert.NotEmpty(t, g.Match("hey everyone"))
}

func TestGreeter_Match_NoMatch(t *testing.T) {
	g := agent.NewGreeter()

	nonGreetings := []string{"build", "test", "run", "deploy", "list"}
	for _, word := range nonGreetings {
		t.Run(word, func(t *testing.T) {
			response := g.Match(word)
			assert.Empty(t, response, "unexpected response for %q", word)
		})
	}
}

func TestGreeter_LearnGreeting(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755)
	require.NoError(t, err)

	g := agent.NewGreeter()

	// Learn a new greeting
	word, learned := g.LearnGreeting("ahoy is a greeting")
	assert.True(t, learned)
	assert.Equal(t, "ahoy", word)

	// Should now match
	assert.NotEmpty(t, g.Match("ahoy"))
}

func TestGreeter_LearnGreeting_AlreadyKnown(t *testing.T) {
	g := agent.NewGreeter()

	// "hello" is already known
	word, learned := g.LearnGreeting("hello is a greeting")
	assert.False(t, learned)
	assert.Equal(t, "hello", word)
}

func TestGreeter_LearnGreeting_InvalidPattern(t *testing.T) {
	g := agent.NewGreeter()

	_, learned := g.LearnGreeting("random text")
	assert.False(t, learned)
}

func TestMatchFortune_Patterns(t *testing.T) {
	patterns := []string{
		"fortune",
		"give me a fortune",
		"show me a fortune",
		"my fortune",
		"feeling lucky",
		"inspire me",
		"motivate me",
		"motivation",
		"wisdom",
		"quote",
	}

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			assert.True(t, agent.MatchFortune(pattern))
		})
	}
}

func TestMatchFortune_NoMatch(t *testing.T) {
	nonFortune := []string{
		"build",
		"test",
		"run",
		"list",
		"help",
		"quit",
	}

	for _, word := range nonFortune {
		t.Run(word, func(t *testing.T) {
			assert.False(t, agent.MatchFortune(word))
		})
	}
}

func TestFortune(t *testing.T) {
	// Fortune should return a non-empty string
	fortune := agent.Fortune()
	assert.NotEmpty(t, fortune)
}
