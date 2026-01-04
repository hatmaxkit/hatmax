package model

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	now := Now()

	if now.IsZero() {
		t.Error("expected Now to return non-zero time")
	}

	if now.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", now.Location())
	}
}

func TestSetCreated(t *testing.T) {
	var createdAt, updatedAt time.Time

	SetCreated(&createdAt, &updatedAt)

	if createdAt.IsZero() {
		t.Error("expected createdAt to be set")
	}

	if updatedAt.IsZero() {
		t.Error("expected updatedAt to be set")
	}

	if createdAt != updatedAt {
		t.Error("expected createdAt and updatedAt to be equal")
	}

	if createdAt.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", createdAt.Location())
	}
}

func TestSetUpdated(t *testing.T) {
	var updatedAt time.Time

	SetUpdated(&updatedAt)

	if updatedAt.IsZero() {
		t.Error("expected updatedAt to be set")
	}

	if updatedAt.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", updatedAt.Location())
	}
}

func TestSetUpdatedModifiesTime(t *testing.T) {
	var updatedAt time.Time
	SetUpdated(&updatedAt)
	firstUpdate := updatedAt

	time.Sleep(10 * time.Millisecond)

	SetUpdated(&updatedAt)
	secondUpdate := updatedAt

	if !secondUpdate.After(firstUpdate) {
		t.Error("expected second update to be after first update")
	}
}
