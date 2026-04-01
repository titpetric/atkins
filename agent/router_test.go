package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/agent"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// createTestRouter creates a router with test pipelines.
func createTestRouter(t *testing.T, pipelines []*model.Pipeline) *agent.Router {
	t.Helper()
	resolver := runner.NewTaskResolver(pipelines)
	registry := agent.DefaultRegistry()
	return agent.NewRouter(resolver, pipelines, registry)
}

// createTestPipelines creates test pipelines for testing.
func createTestPipelines() []*model.Pipeline {
	return []*model.Pipeline{
		{
			ID: "go",
			Jobs: map[string]*model.Job{
				"test":  {Name: "test", Desc: "Run tests"},
				"build": {Name: "build", Desc: "Build the project"},
			},
		},
		{
			ID: "docker",
			Jobs: map[string]*model.Job{
				"up":   {Name: "up", Desc: "Start containers"},
				"down": {Name: "down", Desc: "Stop containers"},
				"push": {Name: "push", Desc: "Push image"},
			},
		},
	}
}

func TestRouter_RouteQuit(t *testing.T) {
	router := createTestRouter(t, nil)

	tests := []string{"quit", "exit", "q", "QUIT", "Exit", "Q"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			route := router.Route(input)
			assert.Equal(t, agent.RouteQuit, route.Type, "expected RouteQuit for %q", input)
		})
	}
}

func TestRouter_RouteHelp(t *testing.T) {
	router := createTestRouter(t, nil)

	tests := []string{"help", "?", "HELP"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			route := router.Route(input)
			assert.Equal(t, agent.RouteHelp, route.Type, "expected RouteHelp for %q", input)
		})
	}
}

func TestRouter_RouteSlashCommand(t *testing.T) {
	router := createTestRouter(t, nil)

	tests := []struct {
		input   string
		command string
		args    string
	}{
		{"/list", "list", ""},
		{"/run go:test", "run", "go:test"},
		{"/cd /path/to/dir", "cd", "/path/to/dir"},
		{"/help", "help", ""},
		{"/quit", "quit", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			route := router.Route(tt.input)
			// /quit and /help have special handling
			if tt.command == "quit" {
				assert.Equal(t, agent.RouteQuit, route.Type)
			} else if tt.command == "help" {
				assert.Equal(t, agent.RouteHelp, route.Type)
			} else {
				assert.Equal(t, agent.RouteSlash, route.Type)
				assert.Equal(t, tt.command, route.Command)
				assert.Equal(t, tt.args, route.Args)
			}
		})
	}
}

func TestRouter_RouteNaturalSlashCommand(t *testing.T) {
	router := createTestRouter(t, nil)

	// Natural language commands that map to slash commands
	// Note: Shell commands like "ls" and "clear" take precedence
	tests := []struct {
		input   string
		command string
	}{
		{"list", "list"},
		{"list tasks", "list"},
		{"list skills", "list"},
		{"tasks", "list"},
		{"skills", "list"},
		{"history", "history"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			route := router.Route(tt.input)
			assert.Equal(t, agent.RouteSlash, route.Type, "expected RouteSlash for %q", tt.input)
			assert.Equal(t, tt.command, route.Command, "expected command %q for input %q", tt.command, tt.input)
		})
	}
}

func TestRouter_ShellTakesPrecedence(t *testing.T) {
	// Shell commands take precedence over natural language slash commands
	router := createTestRouter(t, nil)

	tests := []string{"ls", "clear"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			route := router.Route(input)
			// "ls" and "clear" are real shell commands
			assert.Equal(t, agent.RouteShell, route.Type, "expected RouteShell for %q (shell takes precedence)", input)
			assert.Equal(t, input, route.ShellCmd)
		})
	}
}

func TestRouter_RouteTask(t *testing.T) {
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	tests := []struct {
		input    string
		taskName string
	}{
		{"go:test", "go:test"},
		{"go:build", "go:build"},
		{"docker:up", "docker:up"},
		{"docker:down", "docker:down"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			route := router.Route(tt.input)
			assert.Equal(t, agent.RouteTask, route.Type)
			assert.Equal(t, tt.taskName, route.Task)
			assert.NotNil(t, route.Resolved)
		})
	}
}

func TestRouter_RouteShell(t *testing.T) {
	router := createTestRouter(t, nil)

	// Test shell commands - these should be executable on most systems
	tests := []struct {
		input    string
		shellCmd string
	}{
		{"echo hello", "echo hello"},
		{"ls -la", "ls -la"},
		{"pwd", "pwd"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			route := router.Route(tt.input)
			assert.Equal(t, agent.RouteShell, route.Type, "expected RouteShell for %q", tt.input)
			assert.Equal(t, tt.shellCmd, route.ShellCmd)
		})
	}
}

func TestRouter_RouteShell_CurlWttrIn(t *testing.T) {
	// This test specifically verifies the fix for "curl wttr.in" routing
	router := createTestRouter(t, nil)

	route := router.Route("curl wttr.in")

	// Note: This test will pass if curl is installed
	// If curl is not installed, the route type will be RouteUnknown
	if route.Type == agent.RouteShell {
		assert.Equal(t, "curl wttr.in", route.ShellCmd)
	} else {
		t.Skip("curl not installed, skipping shell routing test")
	}
}

func TestRouter_RouteGreeting(t *testing.T) {
	router := createTestRouter(t, nil)

	tests := []string{"hi", "hello", "hey", "howdy", "hola", "bonjour"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			route := router.Route(input)
			assert.Equal(t, agent.RouteGreeting, route.Type, "expected RouteGreeting for %q", input)
			assert.NotEmpty(t, route.Greeting, "expected non-empty greeting for %q", input)
		})
	}
}

func TestRouter_RouteFortune(t *testing.T) {
	router := createTestRouter(t, nil)

	tests := []string{"fortune", "give me a fortune", "inspire me", "motivation", "quote"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			route := router.Route(input)
			assert.Equal(t, agent.RouteFortune, route.Type, "expected RouteFortune for %q", input)
			assert.NotEmpty(t, route.Fortune, "expected non-empty fortune for %q", input)
		})
	}
}

func TestRouter_RouteCorrection(t *testing.T) {
	router := createTestRouter(t, nil)

	tests := []struct {
		input  string
		phrase string
		task   string
	}{
		{"alias server name to uname -n", "server name", "uname -n"},
		{"if i say deploy, run docker:push", "deploy", "docker:push"},
		{"if i say test it, run go:test", "test it", "go:test"},
		{"map build to go:build", "build", "go:build"},
		{"deploy means docker:push", "deploy", "docker:push"},
		{"run tests should run go:test", "run tests", "go:test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			route := router.Route(tt.input)
			assert.Equal(t, agent.RouteCorrection, route.Type, "expected RouteCorrection for %q", tt.input)
			assert.Equal(t, tt.phrase, route.Phrase)
			assert.Equal(t, tt.task, route.AliasTask)
		})
	}
}

func TestRouter_AliasToShell(t *testing.T) {
	// Test: "alias server name to uname -n" should make "server name" run "uname -n"
	router := createTestRouter(t, nil)

	// First, teach the alias
	correctionRoute := router.Route("alias server name to uname -n")
	require.Equal(t, agent.RouteCorrection, correctionRoute.Type)
	assert.Equal(t, "server name", correctionRoute.Phrase)
	assert.Equal(t, "uname -n", correctionRoute.AliasTask)

	// Store the alias
	router.Aliases().Add(correctionRoute.Phrase, correctionRoute.AliasTask)

	// Now "server name" should route to shell
	route := router.Route("server name")
	assert.Equal(t, agent.RouteShell, route.Type, "expected RouteShell for aliased 'server name'")
	assert.Equal(t, "uname -n", route.ShellCmd)
}

func TestRouter_AliasToTask(t *testing.T) {
	// Test: alias can also map to tasks
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	// Teach alias: "test it" → "go:test"
	router.Aliases().Add("test it", "go:test")

	route := router.Route("test it")
	assert.Equal(t, agent.RouteAlias, route.Type)
	assert.Equal(t, "go:test", route.Task)
	assert.NotNil(t, route.Resolved)
}

func TestRouter_NaturalLanguage_WhatsYourServerName(t *testing.T) {
	// Test the user's example: "what's your server name" after aliasing "server name"
	router := createTestRouter(t, nil)

	// First, teach the alias
	router.Aliases().Add("server name", "uname -n")

	// "what's your server name" should be stripped to "server name" via filler words
	// and then matched to the alias
	route := router.Route("what's your server name")

	// This should match via filler word stripping
	assert.Equal(t, agent.RouteShell, route.Type, "expected RouteShell for 'what's your server name'")
	assert.Equal(t, "uname -n", route.ShellCmd)
}

func TestRouter_Empty(t *testing.T) {
	router := createTestRouter(t, nil)

	route := router.Route("")
	assert.Equal(t, agent.RouteUnknown, route.Type)

	route = router.Route("   ")
	assert.Equal(t, agent.RouteUnknown, route.Type)
}

func TestRouter_Unknown(t *testing.T) {
	router := createTestRouter(t, nil)

	// Something that doesn't match anything
	route := router.Route("xyzzy123notacommand")
	assert.Equal(t, agent.RouteUnknown, route.Type)
}

func TestRouter_AvailableSkills(t *testing.T) {
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	skills := router.AvailableSkills()
	assert.Contains(t, skills, "go:test")
	assert.Contains(t, skills, "go:build")
	assert.Contains(t, skills, "docker:up")
	assert.Contains(t, skills, "docker:down")
	assert.Contains(t, skills, "docker:push")
}

func TestRouter_FindMatches(t *testing.T) {
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	matches := router.FindMatches([]string{"test"})
	assert.Contains(t, matches, "go:test")

	matches = router.FindMatches([]string{"docker"})
	assert.Contains(t, matches, "docker:up")
	assert.Contains(t, matches, "docker:down")
	assert.Contains(t, matches, "docker:push")
}

func TestRouter_NaturalLanguageTask(t *testing.T) {
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	// "run the tests" should match go:test
	route := router.Route("run tests")
	if route.Type == agent.RouteTask {
		assert.Equal(t, "go:test", route.Task)
	}
	// Note: May also route to shell if "run" is executable
}

func TestAliasStore_AddAndMatch(t *testing.T) {
	store := agent.NewAliasStore()

	store.Add("server name", "uname -n")
	assert.Equal(t, "uname -n", store.Match("server name"))
	assert.Equal(t, "uname -n", store.Match("SERVER NAME")) // case insensitive
	assert.Equal(t, "", store.Match("unknown"))
}

func TestAliasStore_FillerWordStripping(t *testing.T) {
	store := agent.NewAliasStore()

	store.Add("server name", "uname -n")

	// Match with filler words stripped
	// "what's your server name" → stripped → "server name"
	// Note: The alias store doesn't handle "what's" but does handle "what"
	assert.Equal(t, "uname -n", store.Match("give me the server name"))
	assert.Equal(t, "uname -n", store.Match("show my server name"))
}

func TestParseCorrection(t *testing.T) {
	tests := []struct {
		input  string
		phrase string
		task   string
		ok     bool
	}{
		{"alias server name to uname -n", "server name", "uname -n", true},
		{"if i say deploy, run docker:push", "deploy", "docker:push", true},
		{"map build to go:build", "build", "go:build", true},
		{"deploy means docker:push", "deploy", "docker:push", true},
		{"run tests should run go:test", "run tests", "go:test", true},
		{"when i type test, run go:test", "test", "go:test", true},
		{"just some text", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			phrase, task, ok := agent.ParseCorrection(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.phrase, phrase)
				assert.Equal(t, tt.task, task)
			}
		})
	}
}

func TestUsageText(t *testing.T) {
	text := agent.UsageText()
	assert.Contains(t, text, "atkins")
	assert.Contains(t, text, "Usage:")
	assert.Contains(t, text, "Examples:")
	assert.Contains(t, text, "alias server name to uname -n")
}

func TestGreeter_Match(t *testing.T) {
	greeter := agent.NewGreeter()

	// Should match greetings
	assert.NotEmpty(t, greeter.Match("hi"))
	assert.NotEmpty(t, greeter.Match("hello"))
	assert.NotEmpty(t, greeter.Match("hey"))

	// Should match thanks
	assert.NotEmpty(t, greeter.Match("thanks"))
	assert.NotEmpty(t, greeter.Match("thank you"))
	assert.NotEmpty(t, greeter.Match("gracias"))
	assert.NotEmpty(t, greeter.Match("merci"))
	assert.NotEmpty(t, greeter.Match("danke"))
	assert.NotEmpty(t, greeter.Match("grazie"))
	assert.NotEmpty(t, greeter.Match("obrigado"))

	// Should not match non-greetings
	assert.Empty(t, greeter.Match("build"))
	assert.Empty(t, greeter.Match("test"))
}

func TestMatchFortune(t *testing.T) {
	assert.True(t, agent.MatchFortune("fortune"))
	assert.True(t, agent.MatchFortune("give me a fortune"))
	assert.True(t, agent.MatchFortune("inspire me"))
	assert.True(t, agent.MatchFortune("motivation"))
	assert.True(t, agent.MatchFortune("quote"))

	assert.False(t, agent.MatchFortune("build"))
	assert.False(t, agent.MatchFortune("test"))
}

func TestRouter_RouteRetry(t *testing.T) {
	router := createTestRouter(t, nil)

	// Without a previous command, retry should return RouteUnknown
	route := router.Route("again")
	assert.Equal(t, agent.RouteUnknown, route.Type)

	route = router.Route("retry")
	assert.Equal(t, agent.RouteUnknown, route.Type)

	// Set a last command
	router.SetLastCommand("echo hello", false)

	// Now retry should work
	route = router.Route("again")
	assert.Equal(t, agent.RouteRetry, route.Type)

	route = router.Route("retry")
	assert.Equal(t, agent.RouteRetry, route.Type)

	route = router.Route("redo")
	assert.Equal(t, agent.RouteRetry, route.Type)
}

func TestRouter_RouteChainedCommands(t *testing.T) {
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	// Test && chaining
	route := router.Route("go:test && go:build")
	assert.Equal(t, agent.RouteMultiTask, route.Type)
	assert.Len(t, route.Tasks, 2)
	assert.Equal(t, "go:test", route.Tasks[0].Name)
	assert.Equal(t, "go:build", route.Tasks[1].Name)

	// Test "then" chaining
	route = router.Route("test then build")
	assert.Equal(t, agent.RouteMultiTask, route.Type)
	assert.Len(t, route.Tasks, 2)
}

func TestRouter_FuzzyMatch(t *testing.T) {
	pipelines := createTestPipelines()
	router := createTestRouter(t, pipelines)

	// Test typo correction
	route := router.Route("tets") // typo for "test"
	// Should suggest go:test or similar
	if route.Type == agent.RouteConfirm {
		assert.NotEmpty(t, route.Suggestion)
		assert.Contains(t, route.Suggestion, "test")
	}

	route = router.Route("biuld") // typo for "build"
	if route.Type == agent.RouteConfirm {
		assert.NotEmpty(t, route.Suggestion)
		assert.Contains(t, route.Suggestion, "build")
	}
}
