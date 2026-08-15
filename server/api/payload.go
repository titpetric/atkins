package api

import (
	"encoding/json"
	"strings"
)

// checkoutRef picks the ref a job should record.
//
// One field decides what gets checked out. Branch and revision were two
// fields for that one decision, which is how a tag ended up travelling
// in the branch field and working only by accident; they are still
// accepted so an existing trigger script keeps working, with the more
// specific of the two winning.
func checkoutRef(ref, revision, branch string) string {
	for _, candidate := range []string{ref, revision, branch} {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			return candidate
		}
	}
	return ""
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
