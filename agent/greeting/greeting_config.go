package greeting

// GreetingConfig is the YAML structure for ~/.atkins/greetings.yaml.
type GreetingConfig struct {
	Greetings []GreetingGroup `yaml:"greetings"`
}
