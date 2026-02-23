package settings

import (
	"fmt"
	"strconv"
	"time"
)

// Value represents a stored setting.
type Value struct {
	Key       string
	Raw       string
	UpdatedAt time.Time
}

// Type defines the data type of a setting.
type Type string

const (
	String Type = "string"
	Int    Type = "int"
	Bool   Type = "bool"
	Enum   Type = "enum"
)

// Schema defines constraints for a setting.
type Schema struct {
	Key         string
	Type        Type
	Default     string
	Label       string
	Description string
	Required    bool
	Secret      bool
	Min         *int
	Max         *int
	MaxLength   int
	Options     []string
	Labels      []string
}

// Validate checks if a value satisfies the schema constraints.
func (s Schema) Validate(raw string) error {
	if s.Required && raw == "" {
		return fmt.Errorf("setting %q is required", s.Key)
	}

	if raw == "" {
		return nil
	}

	if s.MaxLength > 0 && len(raw) > s.MaxLength {
		return fmt.Errorf("setting %q: max length is %d", s.Key, s.MaxLength)
	}

	switch s.Type {
	case Bool:
		_, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("setting %q: expected bool", s.Key)
		}
	case Int:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("setting %q: expected int", s.Key)
		}

		if s.Min != nil && v < *s.Min {
			return fmt.Errorf("setting %q: min is %d", s.Key, *s.Min)
		}

		if s.Max != nil && v > *s.Max {
			return fmt.Errorf("setting %q: max is %d", s.Key, *s.Max)
		}
	case Enum:
		for _, opt := range s.Options {
			if opt == raw {
				return nil
			}
		}

		return fmt.Errorf("setting %q: must be one of %v", s.Key, s.Options)
	}

	return nil
}

// DisplayLabel returns the label or falls back to the key.
func (s Schema) DisplayLabel() string {
	if s.Label != "" {
		return s.Label
	}

	return s.Key
}
