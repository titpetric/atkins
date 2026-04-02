package aliases

import (
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/agent/model"
)

// AliasEntry maps a natural language phrase to a prompt.
// The prompt can be a shell command, prompt target, or any input.
type AliasEntry struct {
	Phrase string `yaml:"phrase"`
	Prompt string `yaml:"prompt"`
}

// Aliases is the alias store type alias for convenience.
type Aliases = AliasStore

// AliasStore manages user-defined phrase to prompt mappings.
type AliasStore struct {
	Aliases []AliasEntry `yaml:"aliases"`

	path string
}

// NewAliasStore loads or creates the alias store.
func NewAliasStore() *AliasStore {
	s := &AliasStore{
		path: aliasStorePath(),
	}
	if s.path != "" {
		s.load()
	}
	return s
}

// Add records a new alias mapping.
func (s *AliasStore) Add(phrase, prompt string) {
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	prompt = strings.TrimSpace(prompt)

	// Update existing alias for the same phrase
	for i, a := range s.Aliases {
		if a.Phrase == phrase {
			s.Aliases[i].Prompt = prompt
			s.save()
			return
		}
	}

	s.Aliases = append(s.Aliases, AliasEntry{Phrase: phrase, Prompt: prompt})
	s.save()
}

// Match checks if the input matches any alias phrase.
// Returns the target task name, or empty string if no match.
func (s *AliasStore) Match(input string) string {
	lower := strings.TrimSpace(input)
	lower = strings.ToLower(lower)
	lower = strings.TrimRight(lower, "!?.,;:-")

	// Exact match first
	for _, a := range s.Aliases {
		if strings.Contains(lower, a.Phrase) {
			return a.Prompt
		}
	}

	// Match with filler words stripped
	cleaned := model.StripFillerWords(lower)
	for _, a := range s.Aliases {
		if strings.Contains(cleaned, a.Phrase) {
			return a.Prompt
		}
	}

	return ""
}

// ParseCorrection detects "if I say X, run Y" style corrections.
// Returns (phrase, task, true) if matched.
func ParseCorrection(input string) (string, string, bool) {
	lower := strings.ToLower(strings.TrimSpace(input))

	// Patterns:
	//   "if i say X, run Y"
	//   "if i say to run X, run Y"
	//   "when i say X, run Y"
	//   "map X to Y"
	//   "alias X to Y"
	//   "X should run Y"
	//   "X means Y"
	prefixes := []struct {
		prefix string
		sep    string
	}{
		{"if i say to run ", ", run "},
		{"if i say ", ", run "},
		{"if i type ", ", run "},
		{"when i say ", ", run "},
		{"when i type ", ", run "},
		{"map ", " to "},
		{"alias ", " to "},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(lower, p.prefix) {
			rest := lower[len(p.prefix):]
			parts := strings.SplitN(rest, p.sep, 2)
			if len(parts) == 2 {
				phrase := strings.TrimSpace(parts[0])
				task := strings.TrimSpace(parts[1])
				// Strip quotes from phrase
				phrase = strings.Trim(phrase, "\"'`")
				task = strings.Trim(task, "\"'`")
				if phrase != "" && task != "" {
					return phrase, task, true
				}
			}
		}
	}

	// "X should run Y"
	if idx := strings.Index(lower, " should run "); idx > 0 {
		phrase := strings.TrimSpace(lower[:idx])
		task := strings.TrimSpace(lower[idx+len(" should run "):])
		phrase = strings.Trim(phrase, "\"'`")
		task = strings.Trim(task, "\"'`")
		if phrase != "" && task != "" {
			return phrase, task, true
		}
	}

	// "X means Y"
	if idx := strings.Index(lower, " means "); idx > 0 {
		phrase := strings.TrimSpace(lower[:idx])
		task := strings.TrimSpace(lower[idx+len(" means "):])
		phrase = strings.Trim(phrase, "\"'`")
		task = strings.Trim(task, "\"'`")
		if phrase != "" && task != "" {
			return phrase, task, true
		}
	}

	return "", "", false
}

func (s *AliasStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = yaml.Unmarshal(data, s)
}

func (s *AliasStore) save() {
	if s.path == "" {
		return
	}
	dir := filepath.Dir(s.path)
	_ = os.MkdirAll(dir, 0o755)

	data, err := yaml.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0o644)
}

func aliasStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".atkins", "aliases.yaml")
}
