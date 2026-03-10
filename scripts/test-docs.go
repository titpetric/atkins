//go:build ignore

// test-docs.go validates documentation examples by:
// 1. Discovering YAML files referenced in @file directives in markdown files
// 2. Discovering YAML files in adjacent folders (article.md -> article/*.yml)
// 3. Running atkins --lint on each discovered file
// 4. Skipping "before" examples (taskfile-before.yml, workflow-before.yml)
//
// Usage: go run scripts/test-docs.go
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const docsContentDir = "docs/content"

// fileDirectivePattern matches @file "Label" path/to/file.yml
var fileDirectivePattern = regexp.MustCompile(`@file\s+"[^"]+"\s+(\S+\.ya?ml)`)

// beforeFilePatterns are files to skip (not valid atkins files)
var beforeFilePatterns = []string{
	"taskfile-before.yml",
	"workflow-before.yml",
	"theme.yml",
}

type result struct {
	file    string
	passed  bool
	skipped bool
	output  string
}

func main() {
	// Find project root (where go.mod lives)
	rootDir, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	docsDir := filepath.Join(rootDir, docsContentDir)

	// Count markdown files scanned
	mdFiles := make(map[string]bool)

	// Collect all YAML files to validate
	yamlFiles := make(map[string]bool)

	// 1. Discover YAML files from @file directives in markdown files
	err = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		mdFiles[path] = true

		files, err := extractFileDirectives(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		mdDir := filepath.Dir(path)
		for _, f := range files {
			absPath := filepath.Join(mdDir, f)
			yamlFiles[absPath] = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking docs directory: %v\n", err)
		os.Exit(1)
	}

	// 2. Discover YAML files from adjacent folders
	err = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// For article.md, check if article/ folder exists
		mdName := strings.TrimSuffix(filepath.Base(path), ".md")
		adjacentDir := filepath.Join(filepath.Dir(path), mdName)

		if info, err := os.Stat(adjacentDir); err == nil && info.IsDir() {
			// Find all .yml files in the adjacent folder (recursively)
			err := filepath.WalkDir(adjacentDir, func(ymlPath string, yd fs.DirEntry, yerr error) error {
				if yerr != nil {
					return yerr
				}
				if !yd.IsDir() && (strings.HasSuffix(ymlPath, ".yml") || strings.HasSuffix(ymlPath, ".yaml")) {
					yamlFiles[ymlPath] = true
				}
				return nil
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to walk %s: %v\n", adjacentDir, err)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking docs directory for adjacent folders: %v\n", err)
		os.Exit(1)
	}

	// Sort files for consistent output
	var sortedFiles []string
	for f := range yamlFiles {
		sortedFiles = append(sortedFiles, f)
	}
	sort.Strings(sortedFiles)

	// Validate each file
	var results []result
	var passCount, failCount, skipCount int

	for _, f := range sortedFiles {
		relPath, _ := filepath.Rel(rootDir, f)

		// Check if file exists
		if _, err := os.Stat(f); os.IsNotExist(err) {
			results = append(results, result{
				file:   relPath,
				passed: false,
				output: "file not found",
			})
			failCount++
			continue
		}

		// Check if this is a "before" example to skip
		if shouldSkip(f) {
			results = append(results, result{
				file:    relPath,
				passed:  true,
				skipped: true,
			})
			skipCount++
			continue
		}

		// Run atkins --lint
		output, passed := runLint(f)
		results = append(results, result{
			file:   relPath,
			passed: passed,
			output: output,
		})
		if passed {
			passCount++
		} else {
			failCount++
		}
	}

	// Print errors first
	for _, r := range results {
		if !r.passed && !r.skipped {
			fmt.Fprintf(os.Stderr, "error: %s\n", r.file)
			for _, line := range strings.Split(r.output, "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "Usage:") && !strings.HasPrefix(line, "--") && !strings.HasPrefix(line, "-") {
					fmt.Fprintf(os.Stderr, "  %s\n", line)
				}
			}
		}
	}

	// Print summary
	total := passCount + skipCount
	fmt.Printf("scanned %d docs, examples passing %d/%d.\n", len(mdFiles), total, len(sortedFiles))

	if failCount > 0 {
		os.Exit(1)
	}
}

// findProjectRoot walks up from current directory to find go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// extractFileDirectives parses a markdown file and returns paths from @file directives
func extractFileDirectives(mdPath string) ([]string, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, err
	}

	matches := fileDirectivePattern.FindAllSubmatch(content, -1)
	var files []string
	for _, m := range matches {
		if len(m) >= 2 {
			files = append(files, string(m[1]))
		}
	}
	return files, nil
}

// shouldSkip returns true if the file should be skipped (before examples)
func shouldSkip(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range beforeFilePatterns {
		if base == pattern {
			return true
		}
	}
	return false
}

// runLint executes atkins --lint on the given file
func runLint(path string) (output string, passed bool) {
	cmd := exec.Command("atkins", "--lint", "-f", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Combine stdout and stderr for error output
		combined := strings.TrimSpace(stdout.String() + stderr.String())
		if combined == "" {
			combined = err.Error()
		}
		return combined, false
	}
	return "", true
}
