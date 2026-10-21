package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/titpetric/atkins/model"
)

// whenContainsLimit caps how much of a file a when: contains check reads.
const whenContainsLimit = 1 << 20

// WhenMatch is the outcome of evaluating a when: block.
type WhenMatch struct {
	// Dir is the folder holding the matched file, where a skill runs.
	Dir string

	// File is the path the matching pattern resolved to.
	File string
}

// MatchWhen evaluates a when: block from startDir.
//
// An absolute pattern resolves on its own. A relative pattern resolves
// in startDir and then in every parent folder, nearest first, unless
// the block sets here: true, which keeps the search to startDir. In each
// folder MatchWhen tries every pattern, so a closer match wins over
// pattern order. With contains: set, only a regular file whose contents match the
// regular expression counts. An invalid contains: expression never
// matches; the linter reports it.
func MatchWhen(when *model.PipelineWhen, startDir string) (WhenMatch, bool) {
	if when.IsEmpty() {
		return WhenMatch{}, false
	}

	var re *regexp.Regexp
	if when.Contains != "" {
		compiled, err := regexp.Compile(when.Contains)
		if err != nil {
			return WhenMatch{}, false
		}
		re = compiled
	}

	for _, pattern := range when.Files {
		if !filepath.IsAbs(pattern) {
			continue
		}
		if match, ok := matchWhenFile("", pattern, re); ok {
			return WhenMatch{Dir: filepath.Dir(match), File: match}, true
		}
	}

	for current := startDir; ; {
		for _, pattern := range when.Files {
			if filepath.IsAbs(pattern) {
				continue
			}
			if match, ok := matchWhenFile(current, pattern, re); ok {
				return WhenMatch{Dir: current, File: match}, true
			}
		}

		parent := filepath.Dir(current)
		if when.Here || parent == current {
			return WhenMatch{}, false
		}
		current = parent
	}
}

// ValidateWhen reports a when: block that can never match because its
// contains: expression does not compile.
func ValidateWhen(when *model.PipelineWhen) error {
	if when == nil || when.Contains == "" {
		return nil
	}
	if _, err := regexp.Compile(when.Contains); err != nil {
		return fmt.Errorf("when: contains %q: %w", when.Contains, err)
	}
	return nil
}

// matchWhenFile resolves one pattern in dir. Without re it behaves as
// MatchWhenPattern. With re it returns the first regular file the
// pattern resolves to whose contents match re.
func matchWhenFile(dir, pattern string, re *regexp.Regexp) (string, bool) {
	if re == nil {
		return MatchWhenPattern(dir, pattern)
	}

	candidates := []string{filepath.Join(dir, pattern)}
	if strings.ContainsAny(pattern, "*?[") {
		matches, err := filepath.Glob(candidates[0])
		if err != nil {
			return "", false
		}
		candidates = matches
	}

	for _, candidate := range candidates {
		if fileContains(candidate, re) {
			return candidate, true
		}
	}
	return "", false
}

// fileContains reports whether a regular file's contents match re,
// reading at most whenContainsLimit bytes.
func fileContains(path string, re *regexp.Regexp) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, whenContainsLimit))
	if err != nil {
		return false
	}
	return re.Match(data)
}

// ApplyJobWhen evaluates the when: block of every job in pipelines from
// startDir. It marks a job Inactive where its block does not match: the
// listing leaves it out and a run refuses it.
func ApplyJobWhen(pipelines []*model.Pipeline, startDir string) {
	for _, pipeline := range pipelines {
		for _, job := range pipeline.GetJobs() {
			if job == nil || job.When.IsEmpty() {
				continue
			}
			match, ok := MatchWhen(job.When, startDir)
			job.Inactive = !ok
			job.Matched = match.File
		}
	}
}
