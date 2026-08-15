package api

import (
	"encoding/json"
	"path"
	"strings"
)

// cleanWorkingDirectory normalizes a client-reported path into a clean
// path relative to the repository root.
//
// The client sends this and an agent will cd into it, so an absolute
// path or a `..` escape is dropped rather than stored: the value only
// ever means "somewhere inside this checkout".
func cleanWorkingDirectory(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." || dir == "/" {
		return ""
	}

	dir = path.Clean(strings.TrimPrefix(strings.ReplaceAll(dir, "\\", "/"), "/"))
	if dir == "." || dir == ".." || strings.HasPrefix(dir, "../") {
		return ""
	}

	return dir
}

// encodeParams serializes job parameters to the JSON object stored on
// the job. A nil map becomes an empty object rather than null, so
// consumers can always unmarshal into a map.
func encodeParams(params map[string]any) (string, error) {
	if len(params) == 0 {
		return "{}", nil
	}

	encoded, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
