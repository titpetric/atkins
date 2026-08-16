package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/platform/pkg/telemetry"

	"github.com/titpetric/atkins/server/model"
)

// SettingStorage reads and writes runtime configuration.
//
// Values are cached in memory because they are read on the dispatch
// path, which runs on every atkins invocation. The cache is invalidated
// on write, and this process is the only writer.
type SettingStorage struct {
	db *sqlx.DB

	mu    sync.RWMutex
	cache map[string]string
}

// NewSettingStorage returns a SettingStorage backed by the given pool.
func NewSettingStorage(db *sqlx.DB) *SettingStorage {
	return &SettingStorage{db: db}
}

// Load fills the cache from the database.
func (s *SettingStorage) Load(ctx context.Context) error {
	ctx, span := telemetry.StartAuto(ctx, s.Load)
	defer span.End()

	query := `SELECT * FROM ` + model.SettingTable
	rows, err := client(s.db).Select[model.Setting](ctx, query)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Name] = row.Value
	}

	s.mu.Lock()
	s.cache = values
	s.mu.Unlock()

	return nil
}

// Get returns the effective value for a setting: the stored override,
// or the registry default.
func (s *SettingStorage) Get(name string) string {
	s.mu.RLock()
	value, ok := s.cache[name]
	s.mu.RUnlock()

	if ok {
		return value
	}
	if definition, found := model.LookupSetting(name); found {
		return definition.Default
	}
	return ""
}

// Override returns the stored value for a setting, and whether an admin
// has stored one at all.
//
// It is the distinction Get deliberately hides: Get answers "what is
// this setting worth", falling back to the registry default, which is
// the wrong question when a start-up flag configures the same thing.
// A default is not a decision, and it must not outrank one written in
// the configuration file.
func (s *SettingStorage) Override(name string) (string, bool) {
	s.mu.RLock()
	value, ok := s.cache[name]
	s.mu.RUnlock()

	return value, ok
}

// Bool returns a setting parsed as a boolean.
func (s *SettingStorage) Bool(name string) bool {
	parsed, _ := strconv.ParseBool(s.Get(name))
	return parsed
}

// Int returns a setting parsed as a whole number.
func (s *SettingStorage) Int(name string) int64 {
	parsed, _ := strconv.ParseInt(s.Get(name), 10, 64)
	return parsed
}

// Bytes returns a setting parsed as a size, such as 32MB.
func (s *SettingStorage) Bytes(name string) int64 {
	parsed, _ := model.ParseBytes(s.Get(name))
	return parsed
}

// Duration returns a setting parsed as a duration.
func (s *SettingStorage) Duration(name string) time.Duration {
	parsed, _ := time.ParseDuration(s.Get(name))
	return parsed
}

// All returns every setting with its effective value, in registry
// order, so an admin sees the whole surface rather than only what has
// been overridden.
func (s *SettingStorage) All() []SettingValue {
	definitions := model.SettingDefinitions()

	values := make([]SettingValue, 0, len(definitions))
	for _, definition := range definitions {
		s.mu.RLock()
		stored, overridden := s.cache[definition.Name]
		s.mu.RUnlock()

		value := definition.Default
		if overridden {
			value = stored
		}

		values = append(values, SettingValue{
			SettingDefinition: definition,
			Value:             value,
			IsDefault:         !overridden,
		})
	}

	return values
}

// SettingValue is a definition together with its effective value.
type SettingValue struct {
	model.SettingDefinition

	Value     string `json:"value"`
	IsDefault bool   `json:"is_default"`
}

// Set stores an override after validating it against the registry.
func (s *SettingStorage) Set(ctx context.Context, name, value, userID string) error {
	ctx, span := telemetry.StartAuto(ctx, s.Set)
	defer span.End()

	definition, ok := model.LookupSetting(name)
	if !ok {
		return fmt.Errorf("unknown setting %q", name)
	}
	if err := definition.ValidateSetting(value); err != nil {
		return err
	}

	now := time.Now()
	setting := &model.Setting{
		Name:            name,
		Value:           value,
		UpdatedByUserID: userID,
	}
	setting.SetCreatedAt(now)
	setting.SetUpdatedAt(now)

	// REPLACE keeps this a single statement whether or not the setting
	// has been written before.
	if err := client(s.db).Replace(ctx, model.SettingTable, setting); err != nil {
		return fmt.Errorf("save setting: %w", err)
	}

	s.mu.Lock()
	if s.cache == nil {
		s.cache = map[string]string{}
	}
	s.cache[name] = value
	s.mu.Unlock()

	return nil
}

// Reset drops an override, returning the setting to its default.
func (s *SettingStorage) Reset(ctx context.Context, name string) error {
	ctx, span := telemetry.StartAuto(ctx, s.Reset)
	defer span.End()

	if _, ok := model.LookupSetting(name); !ok {
		return fmt.Errorf("unknown setting %q", name)
	}

	query := `DELETE FROM ` + model.SettingTable + ` WHERE name = ?`
	if err := client(s.db).Exec(ctx, query, name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("reset setting: %w", err)
		}
	}

	s.mu.Lock()
	delete(s.cache, name)
	s.mu.Unlock()

	return nil
}
