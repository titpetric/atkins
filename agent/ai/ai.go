// Package ai provides the last-resort LLM fallback for the atkins agent:
// when local routing can't resolve an input to a task, alias or shell
// command, the raw input plus the available pipelines are handed to
// `claude -p` and the JSON response is turned into atkins commands to run
// or a short status message to print.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// Response is the JSON shape expected back from `claude -p`.
type Response struct {
	Cmds    []string `json:"cmds"`
	Message string   `json:"message"`
}

// Available reports whether the claude CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// BuildPrompt builds the prompt sent to `claude -p` for a user input that
// local routing couldn't resolve. It embeds the available pipelines (the
// same listing `atkins -l` prints) and the raw atkins.yml, if one is found
// from workDir.
func BuildPrompt(userInput string, pipelines []*model.Pipeline, workDir string) string {
	list := runner.ListPipelines(pipelines)

	yaml := "(no atkins.yml found)"
	if configPath, _, err := runner.DiscoverConfig(workDir); err == nil && configPath != "" {
		if content, err := os.ReadFile(configPath); err == nil {
			yaml = string(content)
		}
	}

	var b strings.Builder
	b.WriteString("The user writes: ")
	b.WriteString(userInput)
	b.WriteString("\n\nThe pipelines available are:\n\n")
	b.WriteString(list)
	b.WriteString("\nAtkins pipeline: ")
	b.WriteString(yaml)
	b.WriteString("\n\nDesired response: One or more commands to run in json format, ")
	b.WriteString(`{"cmds": ["atkins", "atkins release"]} or whatever the user asked for. `)
	b.WriteString("If the user explicitly asks to create new files or atkins.yml, then do as instructed. ")
	b.WriteString(`Reply with {"message": "what you did"}, the message can contain whitespace but keep it short, `)
	b.WriteString(`e.g. "Added new %s target to atkins.yml, added/updated/moved/ {filename}". `)
	b.WriteString(`Don't list all filenames if you can just state the change folder, e.g. `)
	b.WriteString(`"verb(docs): git commit short title only". `)
	b.WriteString(`If no change is requested, if no job can reasonably be inferred from the request, `)
	b.WriteString(`fall back to {"cmds": ["atkins -l"]} to list available jobs.`)

	return b.String()
}

// Invoke runs `claude -p <prompt>` and parses the JSON object out of its
// output. The prompt is passed as a single argv element, not through a
// shell, so no quoting/escaping is needed regardless of size or content.
func Invoke(ctx context.Context, prompt string) (*Response, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude -p failed: %w\n%s", err, out)
	}

	jsonText, ok := extractJSON(string(out))
	if !ok {
		return nil, fmt.Errorf("claude did not return a JSON object:\n%s", out)
	}

	var resp Response
	if err := json.Unmarshal([]byte(jsonText), &resp); err != nil {
		return nil, fmt.Errorf("could not parse claude's JSON response: %w\n%s", err, jsonText)
	}

	return &resp, nil
}

// extractJSON finds the first balanced {...} object in s, tolerating
// surrounding prose or ```json fences.
func extractJSON(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]

		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}

		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}

	return "", false
}

// ValidateAtkinsCmds parses each cmd into argv and rejects the whole batch
// if any command doesn't invoke atkins itself. This is the safety boundary
// for AI-suggested commands: they are executed as argv (never through a
// shell), so this check is sufficient to stop anything but an atkins
// invocation from ever being run, chained shell metacharacters included.
func ValidateAtkinsCmds(cmds []string) ([][]string, error) {
	argvs := make([][]string, 0, len(cmds))
	for _, cmd := range cmds {
		argv := strings.Fields(cmd)
		if len(argv) == 0 || argv[0] != "atkins" {
			return nil, fmt.Errorf("refusing non-atkins command suggested by AI: %q", cmd)
		}
		argvs = append(argvs, argv)
	}
	return argvs, nil
}
