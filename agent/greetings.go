package agent

import (
	rand "math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// GreetingGroup maps a language/group to its trigger words and responses.
type GreetingGroup struct {
	Keywords  []string `yaml:"keywords"`
	Responses []string `yaml:"responses"`
}

// GreetingsConfig is the YAML structure for ~/.atkins/greetings.yaml.
type GreetingsConfig struct {
	Greetings []GreetingGroup `yaml:"greetings"`
}

// Greeter handles greeting detection and responses.
type Greeter struct {
	groups []GreetingGroup
}

// NewGreeter creates a greeter with built-in defaults merged with user config.
func NewGreeter() *Greeter {
	g := &Greeter{
		groups: defaultGreetingGroups(),
	}
	g.loadUserConfig()
	return g
}

func defaultGreetingGroups() []GreetingGroup {
	return []GreetingGroup{
		{
			Keywords: []string{"hi", "hey", "hello", "howdy", "yo", "sup", "heya"},
			Responses: []string{
				"Hey there! Ready to build something?",
				"Hello! What are we working on today?",
				"Hi! Type /list to see what's available.",
				"Hey! Let's get things done.",
				"Hello! How can I help?",
				"Hi there! What's on the agenda?",
			},
		},
		{
			Keywords: []string{"hola", "buenas", "qué tal", "que tal"},
			Responses: []string{
				"¡Hola! ¿Qué tal? ¿En qué trabajamos hoy?",
				"¡Buenas! ¿Qué necesitas?",
				"¡Hola! Escribe /list para ver lo que hay disponible.",
				"¡Hola! ¿Listo para programar?",
				"¡Buenas! ¿Qué hacemos?",
			},
		},
		{
			Keywords: []string{"bonjour", "salut", "coucou"},
			Responses: []string{
				"Bonjour ! Sur quoi on travaille aujourd'hui ?",
				"Salut ! Tape /list pour voir ce qui est disponible.",
				"Bonjour ! Qu'est-ce qu'on fait ?",
				"Salut ! Prêt à coder ?",
			},
		},
		{
			Keywords: []string{"ciao", "salve"},
			Responses: []string{
				"Ciao! Su cosa lavoriamo oggi?",
				"Ciao! Scrivi /list per vedere cosa c'è.",
				"Salve! Pronti a programmare?",
			},
		},
		{
			Keywords: []string{"hallo", "moin", "servus", "grüß gott"},
			Responses: []string{
				"Hallo! Woran arbeiten wir heute?",
				"Moin! Schreib /list um zu sehen, was verfügbar ist.",
				"Hallo! Bereit zum Coden?",
			},
		},
		{
			Keywords: []string{"olá", "oi", "e aí"},
			Responses: []string{
				"Olá! No que vamos trabalhar hoje?",
				"Oi! Digite /list para ver o que está disponível.",
				"E aí! Bora programar?",
			},
		},
	}
}

// Match checks if the input is a greeting and returns a random response.
// Returns empty string if not a greeting.
func (g *Greeter) Match(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))

	// Strip trailing punctuation for matching
	clean := strings.TrimRight(lower, "!?.,; ")

	for _, group := range g.groups {
		for _, kw := range group.Keywords {
			if clean == kw {
				return group.Responses[rand.IntN(len(group.Responses))]
			}
		}
	}
	return ""
}

// fortunePatterns are words that, when all present in the input, trigger a fortune.
var fortunePatterns = [][]string{
	{"fortune"},
	{"give", "fortune"},
	{"show", "fortune"},
	{"tell", "fortune"},
	{"my", "fortune"},
	{"a", "fortune"},
	{"feeling", "lucky"},
	{"inspire", "me"},
	{"motivate", "me"},
	{"motivation"},
	{"wisdom"},
	{"quote"},
}

// MatchFortune returns true if the input is asking for a fortune.
func MatchFortune(input string) bool {
	lower := strings.ToLower(strings.TrimRight(strings.TrimSpace(input), "!?.,; "))
	words := strings.Fields(lower)
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}

	for _, pattern := range fortunePatterns {
		matched := true
		for _, p := range pattern {
			if !wordSet[p] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// LearnGreeting parses input like "ciao is a greeting" and adds the word.
// Returns true if a new greeting was learned.
func (g *Greeter) LearnGreeting(input string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(input))

	// Match patterns: "X is a greeting", "X is a saludo", etc.
	suffixes := []string{
		" is a greeting",
		" is a salutation",
		" is a saludo",
		" is a gruss",
		" is a gruß",
		" is a salut",
	}

	var word string
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			word = strings.TrimSuffix(lower, suffix)
			word = strings.TrimSpace(word)
			break
		}
	}
	if word == "" {
		return "", false
	}

	// Don't add if already known
	if g.Match(word) != "" {
		return word, false
	}

	// Add to the first (English) group as default
	g.groups[0].Keywords = append(g.groups[0].Keywords, word)
	g.saveUserConfig()
	return word, true
}

func (g *Greeter) loadUserConfig() {
	path := greetingsConfigPath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg GreetingsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return
	}

	// Merge user groups: if keywords overlap with a built-in group, extend it;
	// otherwise append as a new group.
	for _, userGroup := range cfg.Greetings {
		merged := false
		for i := range g.groups {
			if hasOverlap(g.groups[i].Keywords, userGroup.Keywords) {
				g.groups[i].Keywords = mergeUnique(g.groups[i].Keywords, userGroup.Keywords)
				if len(userGroup.Responses) > 0 {
					g.groups[i].Responses = append(g.groups[i].Responses, userGroup.Responses...)
				}
				merged = true
				break
			}
		}
		if !merged && len(userGroup.Keywords) > 0 {
			g.groups = append(g.groups, userGroup)
		}
	}
}

func (g *Greeter) saveUserConfig() {
	path := greetingsConfigPath()
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)

	cfg := GreetingsConfig{Greetings: g.groups}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func greetingsConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".atkins", "greetings.yaml")
}

func hasOverlap(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, s := range a {
		set[s] = true
	}
	for _, s := range b {
		if set[s] {
			return true
		}
	}
	return false
}

func mergeUnique(a, b []string) []string {
	set := make(map[string]bool, len(a))
	for _, s := range a {
		set[s] = true
	}
	for _, s := range b {
		if !set[s] {
			a = append(a, s)
			set[s] = true
		}
	}
	return a
}

// Fortune returns a fortune string. Uses the system `fortune` command if
// available, otherwise returns a random coding motivational quote.
func Fortune() string {
	if path, err := exec.LookPath("fortune"); err == nil {
		if out, err := exec.Command(path, "-s").Output(); err == nil {
			return strings.TrimRight(string(out), "\n")
		}
	}
	return codingFortunes[rand.IntN(len(codingFortunes))]
}

var codingFortunes = []string{
	"The best error message is the one that never shows up.",
	"First, solve the problem. Then, write the code. — John Johnson",
	"Code is like humor. When you have to explain it, it's bad. — Cory House",
	"Make it work, make it right, make it fast. — Kent Beck",
	"Simplicity is the soul of efficiency. — Austin Freeman",
	"Every great developer you know got there by solving problems they were unqualified to solve until they actually did it. — Patrick McKenzie",
	"The most disastrous thing that you can ever learn is your first programming language. — Alan Kay",
	"Programs must be written for people to read, and only incidentally for machines to execute. — Harold Abelson",
	"Any fool can write code that a computer can understand. Good programmers write code that humans can understand. — Martin Fowler",
	"It's not a bug — it's an undocumented feature.",
	"Deleted code is debugged code. — Jeff Sickel",
	"The only way to go fast, is to go well. — Robert C. Martin",
	"A ship in port is safe, but that's not what ships are built for. — Grace Hopper",
	"Talk is cheap. Show me the code. — Linus Torvalds",
	"Weeks of coding can save you hours of planning.",
	"There are only two hard things in Computer Science: cache invalidation and naming things. — Phil Karlton",
	"If debugging is the process of removing bugs, then programming must be the process of putting them in. — Edsger Dijkstra",
	"Perfection is achieved not when there is nothing more to add, but when there is nothing left to take away. — Antoine de Saint-Exupéry",
}
