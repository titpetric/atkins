package runner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// compareFile diffs one source file against its vendored copy.
func (v *Vendorer) compareFile(path string) (status VendorStatus, added, removed int, diff string) {
	target := filepath.Join(v.TargetDir, filepath.Base(path))

	existing, readErr := os.ReadFile(target)

	data, err := os.ReadFile(path)
	if err != nil {
		// Install reports the read error properly; treat it as work to do.
		return VendorChanged, 0, 0, ""
	}

	from, to := diffLines(existing), diffLines(data)

	switch {
	case readErr != nil:
		// Nothing to diff against, so the whole file is the addition.
		return VendorNew, len(to), 0, ""
	case bytes.Equal(existing, data):
		return VendorCurrent, 0, 0, ""
	}

	for _, op := range difflib.NewMatcher(from, to).GetOpCodes() {
		switch op.Tag {
		case 'r':
			removed += op.I2 - op.I1
			added += op.J2 - op.J1
		case 'd':
			removed += op.I2 - op.I1
		case 'i':
			added += op.J2 - op.J1
		}
	}

	fromLabel := target
	if rel, err := filepath.Rel(v.Root, target); err == nil {
		fromLabel = rel
	}

	// A diff that can't be rendered is not worth failing over; the line
	// counts above are what the caller reports either way.
	diff, _ = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        from,
		B:        to,
		FromFile: fromLabel,
		ToFile:   path,
		Context:  3,
	})

	return VendorChanged, added, removed, diff
}

// diffLines splits a file into lines that each keep their newline, with
// no phantom line for the trailing one.
func diffLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	lines := strings.SplitAfter(string(data), "\n")
	if last := len(lines) - 1; lines[last] == "" {
		return lines[:last]
	}

	return lines
}

// install copies one source file into TargetDir under its own name.
func (v *Vendorer) install(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read skill %s: %w", path, err)
	}

	target := filepath.Join(v.TargetDir, filepath.Base(path))
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("failed to write skill %s: %w", target, err)
	}

	return nil
}
