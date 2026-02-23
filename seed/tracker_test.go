package seed

import "testing"

func TestNewTracker(t *testing.T) {
	// Test that NewTracker doesn't panic with nil db
	// (actual DB operations would require integration tests)
	tracker := NewTracker(nil)
	if tracker == nil {
		t.Error("NewTracker returned nil")
	}

	if tracker.db != nil {
		t.Error("tracker.db should be nil when initialized with nil")
	}
}
