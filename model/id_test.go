package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateID(t *testing.T) {
	var id string
	GenerateID(&id)

	if id == "" {
		t.Error("expected ID to be generated")
	}

	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("expected valid UUID, got error: %v", err)
	}
}

func TestNewID(t *testing.T) {
	id := NewID()

	if id == "" {
		t.Error("expected ID to be generated")
	}

	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("expected valid UUID, got error: %v", err)
	}
}

func TestGenerateIDUniqueness(t *testing.T) {
	var id1, id2 string
	GenerateID(&id1)
	GenerateID(&id2)

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestNewIDUniqueness(t *testing.T) {
	id1 := NewID()
	id2 := NewID()

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}
