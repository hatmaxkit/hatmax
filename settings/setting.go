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
	Key      string
	Type     Type
	Default  string
	Required bool
	Min      *int
	Max      *int
	Options  []string
}

// Validate checks if a value satisfies the schema constraints.
func (s Schema) Validate(raw string) error {
	if s.Required && raw == "" {
		return fmt.Errorf("setting %q is required", s.Key)
	}
	if raw == "" {
		return nil
	}

	switch s.Type {
	case Bool:
		if _, err := strconv.ParseBool(raw); err != nil {
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
