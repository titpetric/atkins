package model

import (
	"fmt"
	"strconv"
	"time"
)

// Setting names. These are the configuration an admin can change at
// runtime; everything else is a process flag and needs a restart.
const (
	// SettingRepositoryPolicy is "open" or "allowlist".
	SettingRepositoryPolicy = "repository.policy"

	// SettingRegistrationOpen decides whether anyone may create an
	// account. The first account is always allowed regardless.
	SettingRegistrationOpen = "registration.open"

	// SettingJobMaxDepth bounds how deep a job may dispatch children.
	SettingJobMaxDepth = "job.max_depth"

	// SettingJobLeaseTTL is how long an agent may hold a job without a
	// heartbeat before the server reclaims it.
	SettingJobLeaseTTL = "job.lease_ttl"

	// SettingJobRetention is how long finished jobs and their output
	// are kept. Zero keeps them forever.
	SettingJobRetention = "job.retention"
)

// SettingKind is how a setting's value is validated and parsed.
type SettingKind string

// Setting kinds.
const (
	KindString   SettingKind = "string"
	KindBool     SettingKind = "bool"
	KindInt      SettingKind = "int"
	KindDuration SettingKind = "duration"
	KindEnum     SettingKind = "enum"
)

// SettingDefinition describes one configurable value.
type SettingDefinition struct {
	Name        string      `json:"name"`
	Kind        SettingKind `json:"kind"`
	Default     string      `json:"default"`
	Description string      `json:"description"`

	// Values enumerates the accepted values for KindEnum.
	Values []string `json:"values,omitempty"`
}

// settingDefinitions is the registry, in the order the API lists them.
var settingDefinitions = []SettingDefinition{
	{
		Name:        SettingRepositoryPolicy,
		Kind:        KindEnum,
		Default:     PolicyOpen,
		Values:      []string{PolicyOpen, PolicyAllowlist},
		Description: "Which repositories agents may build: any, or only those matching an allowlist rule.",
	},
	{
		Name:        SettingRegistrationOpen,
		Kind:        KindBool,
		Default:     "false",
		Description: "Allow anyone to register. The first account on an empty instance is always allowed.",
	},
	{
		Name:        SettingJobMaxDepth,
		Kind:        KindInt,
		Default:     "3",
		Description: "How deep a job may dispatch child jobs before the server refuses.",
	},
	{
		Name:        SettingJobLeaseTTL,
		Kind:        KindDuration,
		Default:     "15m",
		Description: "How long an agent may hold a claimed job without a heartbeat.",
	},
	{
		Name:        SettingJobRetention,
		Kind:        KindDuration,
		Default:     "0",
		Description: "How long finished jobs are kept. 0 keeps them forever.",
	},
}

// SettingDefinitions returns the registry.
func SettingDefinitions() []SettingDefinition {
	definitions := make([]SettingDefinition, len(settingDefinitions))
	copy(definitions, settingDefinitions)
	return definitions
}

// LookupSetting returns the definition for a name.
func LookupSetting(name string) (SettingDefinition, bool) {
	for _, definition := range settingDefinitions {
		if definition.Name == name {
			return definition, true
		}
	}
	return SettingDefinition{}, false
}

// ValidateSetting checks a value against its definition.
func (d SettingDefinition) ValidateSetting(value string) error {
	switch d.Kind {
	case KindBool:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%s must be true or false", d.Name)
		}
	case KindInt:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%s must be a whole number", d.Name)
		}
	case KindDuration:
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("%s must be a duration such as 15m or 24h", d.Name)
		}
	case KindEnum:
		for _, allowed := range d.Values {
			if value == allowed {
				return nil
			}
		}
		return fmt.Errorf("%s must be one of %v", d.Name, d.Values)
	}

	return nil
}
