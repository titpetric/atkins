package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ShellHistoryEntry records a shell command execution.
type ShellHistoryEntry struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Dir      string        `json:"dir"`
	Time     time.Time     `json:"time"`
}

// ShellHistory maintains a persistent history of shell commands.
type ShellHistory struct {
	Entries []ShellHistoryEntry `json:"entries"`
	path    string
}

// NewShellHistory loads or creates a shell history file.
func NewShellHistory() *ShellHistory {
	h := &ShellHistory{}
	h.path = shellHistoryPath()
	if h.path != "" {
		h.load()
	}
	return h
}

// Add records a command execution.
func (h *ShellHistory) Add(command string, exitCode int, duration time.Duration, dir string) {
	h.Entries = append(h.Entries, ShellHistoryEntry{
		Command:  command,
		ExitCode: exitCode,
		Duration: duration,
		Dir:      dir,
		Time:     time.Now(),
	})
	h.save()
}

// Match returns shell history entries where the command starts with or
// contains the given input. Results are returned most-recent-first.
func (h *ShellHistory) Match(input string) []ShellHistoryEntry {
	lower := strings.ToLower(input)
	var matches []ShellHistoryEntry

	// Walk backwards for most-recent-first
	seen := make(map[string]bool)
	for i := len(h.Entries) - 1; i >= 0; i-- {
		e := h.Entries[i]
		cmdLower := strings.ToLower(e.Command)
		if seen[e.Command] {
			continue
		}
		if strings.HasPrefix(cmdLower, lower) || strings.Contains(cmdLower, lower) {
			seen[e.Command] = true
			matches = append(matches, e)
		}
		if len(matches) >= 10 {
			break
		}
	}
	return matches
}

// FindExact returns the most recent entry matching the exact command, or nil.
func (h *ShellHistory) FindExact(command string) *ShellHistoryEntry {
	for i := len(h.Entries) - 1; i >= 0; i-- {
		if h.Entries[i].Command == command {
			return &h.Entries[i]
		}
	}
	return nil
}

func (h *ShellHistory) load() {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, h)
}

func (h *ShellHistory) save() {
	if h.path == "" {
		return
	}
	dir := filepath.Dir(h.path)
	_ = os.MkdirAll(dir, 0o755)

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(h.path, data, 0o644)
}

func shellHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".atkins", "shell_history.json")
}
