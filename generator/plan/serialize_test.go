package plan

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarshalYAMLIsStableAndOrdered(t *testing.T) {
	value := validTestPlan()

	first, err := MarshalYAML(value)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	second, err := MarshalYAML(value)
	if err != nil {
		t.Fatalf("second MarshalYAML() error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf("MarshalYAML() output is not stable:\n%s\n---\n%s", first, second)
	}

	text := string(first)
	identityIndex := strings.Index(text, "intent: create_feature")
	domainIndex := strings.Index(text, "domain:")
	operationsIndex := strings.Index(text, "operations:")

	validationIndex := strings.Index(text, "validation:")
	if identityIndex < 0 || domainIndex <= identityIndex || operationsIndex <= domainIndex || validationIndex <= operationsIndex {
		t.Errorf("MarshalYAML() fields are not in contract order:\n%s", text)
	}
}

func TestCanonicalJSONIsStable(t *testing.T) {
	value := validTestPlan()

	first, err := canonicalJSON(value)
	if err != nil {
		t.Fatalf("canonicalJSON() error = %v", err)
	}

	second, err := canonicalJSON(value)
	if err != nil {
		t.Fatalf("second canonicalJSON() error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Errorf("canonicalJSON() output is not stable")
	}
}

func TestMarshalYAMLRejectsInvalidPlan(t *testing.T) {
	value := validTestPlan()
	value.Operations = nil

	_, err := MarshalYAML(value)
	requirePlanCode(t, err, "plan_required_field")
}
