package intent

import (
	"bytes"
	"fmt"
	"io"
	"sort"

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
	normalizeSemanticNames(value)
	normalizeApplication(value)

	if version := canonicalVersion(value.HatmaxVersion); version != "" {
		value.HatmaxVersion = version
	}

	if value.Documentation == "" {
		value.Documentation = DocumentationNotRequested
	}

	if value.Capabilities == nil {
		value.Capabilities = []string{}
	}

	if value.DocumentationTargets == nil {
		value.DocumentationTargets = []DocumentationTarget{}
	} else {
		value.DocumentationTargets = append([]DocumentationTarget{}, value.DocumentationTargets...)
		sort.SliceStable(value.DocumentationTargets, func(left, right int) bool {
			leftTarget := value.DocumentationTargets[left]
			rightTarget := value.DocumentationTargets[right]
			leftRank := documentationQuadrantRank(leftTarget.Quadrant)

			rightRank := documentationQuadrantRank(rightTarget.Quadrant)
			if leftRank != rightRank {
				return leftRank < rightRank
			}

			if leftTarget.Subject != rightTarget.Subject {
				return leftTarget.Subject < rightTarget.Subject
			}

			return leftTarget.ReaderGoal < rightTarget.ReaderGoal
		})
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

	if value.InitialFeatures == nil {
		value.InitialFeatures = []InitialFeature{}
	}
}

func documentationQuadrantRank(value DocumentationQuadrant) int {
	switch value {
	case DocumentationTutorial:
		return 0
	case DocumentationHowTo:
		return 1
	case DocumentationReference:
		return 2
	case DocumentationExplanation:
		return 3
	default:
		return 4
	}
}
