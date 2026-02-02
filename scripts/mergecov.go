package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CoverageBlock represents a covered line in coverage data
type CoverageBlock struct {
	StartLine, EndLine int
	StartCol, EndCol   int
	Stmts, Count       int
	Filename           string
	Covered            bool
	Count2             int // Track if covered by multiple tests
}

// CoverageProfile stores all coverage data keyed by file and line range
type CoverageProfile struct {
	Mode   string
	Blocks map[string]map[string]*CoverageBlock
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mergecov <output-file> [<cov-dir>]\n")
		os.Exit(1)
	}

	outFile := os.Args[1]
	covDir := "./coverage"
	if len(os.Args) > 2 {
		covDir = os.Args[2]
	}

	// Find all .cov files
	covFiles, err := findCovFiles(covDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding coverage files: %v\n", err)
		os.Exit(1)
	}

	if len(covFiles) == 0 {
		fmt.Fprintf(os.Stderr, "no coverage files found in %s\n", covDir)
		os.Exit(1)
	}

	profile := &CoverageProfile{
		Mode:   "set", // merged output is always mode: set
		Blocks: make(map[string]map[string]*CoverageBlock),
	}

	// Parse all coverage files
	for _, covFile := range covFiles {
		if err := parseCoverageFile(covFile, profile); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", covFile, err)
			os.Exit(1)
		}
	}

	// Write merged coverage file
	if err := writeProfile(outFile, profile); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outFile, err)
		os.Exit(1)
	}

	fmt.Printf("Merged %d coverage files into %s\n", len(covFiles), outFile)
}

func findCovFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subFiles, err := findCovFiles(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else if strings.HasSuffix(entry.Name(), ".cov") && entry.Name() != "merged.cov" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}

func parseCoverageFile(filename string, profile *CoverageProfile) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip the mode line
	if !scanner.Scan() {
		return fmt.Errorf("empty coverage file")
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse coverage line: filename:start.col,end.col numStmt count
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		locationParts := strings.Split(parts[0], ":")
		if len(locationParts) != 2 {
			continue
		}

		filename := locationParts[0]
		rangeParts := strings.Split(locationParts[1], ",")
		if len(rangeParts) != 2 {
			continue
		}

		startParts := strings.Split(rangeParts[0], ".")
		endParts := strings.Split(rangeParts[1], ".")
		if len(startParts) != 2 || len(endParts) != 2 {
			continue
		}

		startLine, _ := strconv.Atoi(startParts[0])
		startCol, _ := strconv.Atoi(startParts[1])
		endLine, _ := strconv.Atoi(endParts[0])
		endCol, _ := strconv.Atoi(endParts[1])
		stmts, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		key := fmt.Sprintf("%d.%d,%d.%d,%d", startLine, startCol, endLine, endCol, stmts)

		if profile.Blocks[filename] == nil {
			profile.Blocks[filename] = make(map[string]*CoverageBlock)
		}

		if existing, ok := profile.Blocks[filename][key]; ok {
			// Line already covered, mark as covered by multiple tests if it was covered
			if count > 0 && existing.Count > 0 {
				existing.Count2++
			}
			existing.Covered = existing.Covered || (count > 0)
			if count > existing.Count {
				existing.Count = count
			}
		} else {
			profile.Blocks[filename][key] = &CoverageBlock{
				StartLine: startLine,
				StartCol:  startCol,
				EndLine:   endLine,
				EndCol:    endCol,
				Stmts:     stmts,
				Count:     count,
				Filename:  filename,
				Covered:   count > 0,
			}
		}
	}

	return scanner.Err()
}

func writeProfile(filename string, profile *CoverageProfile) error {
	os.MkdirAll(filepath.Dir(filename), 0o755)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write mode line
	fmt.Fprintf(writer, "mode: %s\n", profile.Mode)

	// Collect and sort all filenames
	var filenames []string
	for fn := range profile.Blocks {
		filenames = append(filenames, fn)
	}
	sort.Strings(filenames)

	// Write coverage data for each file
	for _, fn := range filenames {
		blocks := profile.Blocks[fn]

		// Sort blocks by start line
		var keys []string
		for k := range blocks {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			pi := strings.Split(keys[i], ".")
			pj := strings.Split(keys[j], ".")
			si, _ := strconv.Atoi(pi[0])
			sj, _ := strconv.Atoi(pj[0])
			return si < sj
		})

		for _, k := range keys {
			block := blocks[k]
			count := 0
			if block.Covered {
				count = 1
			}
			fmt.Fprintf(writer, "%s:%d.%d,%d.%d %d %d\n",
				fn, block.StartLine, block.StartCol, block.EndLine, block.EndCol, block.Stmts, count)
		}
	}

	return nil
}
