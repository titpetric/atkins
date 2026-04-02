package greeting

// GreetingGroup maps a language/group to its trigger words and responses.
type GreetingGroup struct {
	Keywords  []string `yaml:"keywords"`
	Responses []string `yaml:"responses"`
}
