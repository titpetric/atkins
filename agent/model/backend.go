package model

// Cmd represents a command to be executed by the runtime.
// This is an interface to decouple from specific implementations (e.g., bubbletea).
type Cmd = func() any

// Model represents the application model interface.
// Implementations provide the actual state and behavior.
type Model interface {
	// Empty interface - implementations define their own methods
}
