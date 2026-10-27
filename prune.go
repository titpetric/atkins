package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/titpetric/atkins/storage"
)

// runPrune runs `atkins --prune`: it removes the runs and commands older
// than --older-than from the database --db names and releases the space,
// or with --dry-run counts them and changes nothing, and prints one
// line: what went, what remains and the file's size before and
// after. A database that is not there is an error, so a typo makes no
// empty file.
func runPrune(ctx context.Context, stdout io.Writer, opts *Options) error {
	if opts.DB == "" {
		return errors.New("--prune needs --db, the database to prune")
	}
	age, err := ParseAge(opts.OlderThan)
	if err != nil {
		return err
	}
	if _, err := os.Stat(opts.DB); err != nil {
		return err
	}
	store, err := storage.NewStorageAt(ctx, opts.DB)
	if err != nil {
		return err
	}
	defer store.Close()

	before := time.Now().Add(-age)
	size := storage.Size(opts.DB)
	gone, err := store.Count(ctx, before)
	if err != nil {
		return err
	}
	all, err := store.Count(ctx, time.Time{})
	if err != nil {
		return err
	}
	verb := "would remove"
	if !opts.DryRun {
		verb = "removed"
		if gone, err = store.Prune(ctx, before); err != nil {
			return err
		}
		if all, err = store.Count(ctx, time.Time{}); err != nil {
			return err
		}
	} else {
		all = &storage.PurgeResult{Runs: all.Runs - gone.Runs, Commands: all.Commands - gone.Commands}
	}
	after := size
	if !opts.DryRun {
		after = storage.Size(opts.DB)
	}
	_, err = fmt.Fprintf(stdout, "%s: %s %s older than %s; %s left; %s before, %s after\n",
		opts.DB, verb, counts(gone), strings.TrimSpace(opts.OlderThan), counts(all), formatSize(size), formatSize(after))
	return err
}

// counts reads a result: "3 runs, 1 command".
func counts(r *storage.PurgeResult) string {
	return plural(r.Runs, "run") + ", " + plural(r.Commands, "command")
}

// ParseAge reads an age as --older-than takes it: an integer count of days,
// `31d`, or a duration Go reads, `720h`. Empty, zero and a negative
// age are errors: a prune of everything is no default.
func ParseAge(text string) (time.Duration, error) {
	text = strings.TrimSpace(text)
	if days, ok := strings.CutSuffix(text, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("--older-than: %q is no age; use days, such as 31d, or a duration, such as 720h", text)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	age, err := time.ParseDuration(text)
	if err != nil || age <= 0 {
		return 0, fmt.Errorf("--older-than: %q is no age; use days, such as 31d, or a duration, such as 720h", text)
	}
	return age, nil
}

// formatSize writes a byte count: 512 B, 1.5 MB.
func formatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value, suffix := float64(n)/unit, "KB"
	for _, next := range []string{"MB", "GB", "TB"} {
		if value < unit {
			break
		}
		value, suffix = value/unit, next
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}
