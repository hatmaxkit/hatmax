package plan

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// MarshalYAML returns a stable user-visible representation of a valid plan.
func MarshalYAML(value Plan) ([]byte, error) {
	err := Validate(value)
	if err != nil {
		return nil, fmt.Errorf("validate plan: %w", err)
	}

	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal plan YAML: %w", err)
	}

	return data, nil
}

func canonicalJSON(value Plan) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical plan: %w", err)
	}

	return data, nil
}
