package model

// Cmd represents a command for the runtime to execute.
// This is an interface to decouple from specific implementations (e.g., bubbletea).
type Cmd = func() any

// Model represents the application model interface.
// Implementations hold the actual state and behavior.
type Model interface {
	// Empty interface - implementations define their own methods
}
