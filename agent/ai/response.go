package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Response is the JSON shape expected back from `claude -p`.
type Response struct {
	Cmds    []string `json:"cmds"`
	Message string   `json:"message"`
}

// Invoke runs `claude -p <prompt>` and parses its stdout. The prompt is one
// argv element rather than a shell word, so it needs no quoting whatever it
// contains. Only stdout is parsed: claude writes progress and warnings to
// stderr, and a brace in either would otherwise reach the JSON scan.
func Invoke(ctx context.Context, prompt string) (*Response, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude -p failed: %w\n%s", err, strings.TrimSpace(stderr.String()))
	}

	return ParseResponse(stdout.String())
}

// ParseResponse reads the first JSON object out of claude's output.
func ParseResponse(out string) (*Response, error) {
	jsonText, ok := extractJSON(out)
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
// surrounding prose or a ```json fence.
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
