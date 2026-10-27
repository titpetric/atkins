package storage

import (
	"context"
	"os"
	"time"
)

// Prune removes every run and command started before a time, batch
// after batch until none remains, then releases the space: it vacuums
// the file and truncates its write-ahead log. A zero time removes
// nothing.
func (s *Storage) Prune(ctx context.Context, before time.Time) (*PurgeResult, error) {
	total := &PurgeResult{}
	for {
		result, err := s.Purge(ctx, PurgeRequest{Before: before})
		if err != nil {
			return nil, err
		}
		total.Runs += result.Runs
		total.Commands += result.Commands
		if result.Runs == 0 && result.Commands == 0 {
			break
		}
	}
	if total.Runs == 0 && total.Commands == 0 {
		return total, nil
	}
	if _, err := s.db.ExecContext(ctx, "VACUUM"); err != nil {
		return nil, err
	}
	_, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	return total, err
}

// Size returns the bytes a database takes with its wal and shm files, zero when absent.
func Size(path string) int64 {
	var size int64
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if info, err := os.Stat(path + suffix); err == nil {
			size += info.Size()
		}
	}
	return size
}
