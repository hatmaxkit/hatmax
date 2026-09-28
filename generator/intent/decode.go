package intent

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// DecodeYAML decodes, normalizes, and structurally validates one intent.
func DecodeYAML(data []byte) (Intent, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var result Intent

	err := decoder.Decode(&result)
	if err != nil {
		return Intent{}, fmt.Errorf("decode intent: %w", err)
	}

	var extra any

	err = decoder.Decode(&extra)
	if err != io.EOF {
		if err != nil {
			return Intent{}, fmt.Errorf("decode trailing intent content: %w", err)
		}

		return Intent{}, fmt.Errorf("decode intent: multiple YAML documents are not allowed")
	}

	normalize(&result)

	err = ValidateSchema(result)
	if err != nil {
		return Intent{}, err
	}

	return result, nil
}

func normalize(value *Intent) {
	if version := canonicalVersion(value.HatmaxVersion); version != "" {
		value.HatmaxVersion = version
	}

	if value.Documentation == "" {
		value.Documentation = DocumentationNotRequested
	}

	if value.Capabilities == nil {
		value.Capabilities = []string{}
	}

	if value.Exceptions == nil {
		value.Exceptions = []Exception{}
	}

	if value.Domain.Fields == nil {
		value.Domain.Fields = []Field{}
	}

	if value.Domain.Rules == nil {
		value.Domain.Rules = []BusinessRule{}
	}
}
