package eventlog

import (
	"os"
	"path/filepath"
	"strings"
)

// CaptureModulePath captures the Go module path from go.mod if present.
func CaptureModulePath() string {
	// Look for go.mod in current directory and parents
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		modPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modPath); err == nil {
			// Parse module line
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module"))
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}
