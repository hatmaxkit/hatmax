package fake

import (
	"sync"
	"testing"
)

func TestCounterIncrementRequests(t *testing.T) {
	c := NewCounter()

	c.IncrementRequests()
	c.IncrementRequests()
	c.IncrementRequests()

	if c.IncrementCount() != 3 {
		t.Errorf("IncrementCount() = %d, want 3", c.IncrementCount())
	}
}

func TestCounterGetAndResetRequests(t *testing.T) {
	c := NewCounter()

	c.IncrementRequests()
	c.IncrementRequests()

	got := c.GetAndResetRequests()
	if got != 2 {
		t.Errorf("GetAndResetRequests() = %d, want 2", got)
	}

	got = c.GetAndResetRequests()
	if got != 0 {
		t.Errorf("second GetAndResetRequests() = %d, want 0", got)
	}

	if c.GetAndResetCount() != 2 {
		t.Errorf("GetAndResetCount() = %d, want 2", c.GetAndResetCount())
	}
}

func TestCounterWithCount(t *testing.T) {
	c := NewCounter().WithCount(100)

	got := c.GetAndResetRequests()
	if got != 100 {
		t.Errorf("GetAndResetRequests() = %d, want 100", got)
	}
}

func TestCounterReset(t *testing.T) {
	c := NewCounter()
	c.IncrementRequests()
	c.GetAndResetRequests()

	c.Reset()

	if c.IncrementCount() != 0 {
		t.Errorf("after Reset, IncrementCount() = %d, want 0", c.IncrementCount())
	}

	if c.GetAndResetCount() != 0 {
		t.Errorf("after Reset, GetAndResetCount() = %d, want 0", c.GetAndResetCount())
	}
}

func TestCounterCustomIncrementFunc(t *testing.T) {
	called := false
	c := NewCounter()
	c.IncrementFunc = func() {
		called = true
	}

	c.IncrementRequests()

	if !called {
		t.Error("IncrementFunc was not called")
	}
}

func TestCounterCustomGetAndResetFunc(t *testing.T) {
	c := NewCounter()
	c.GetAndResetFunc = func() int64 {
		return 999
	}

	got := c.GetAndResetRequests()
	if got != 999 {
		t.Errorf("GetAndResetRequests() = %d, want 999", got)
	}
}

func TestCounterConcurrent(t *testing.T) {
	c := NewCounter()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			c.IncrementRequests()
		}()
	}

	wg.Wait()

	if c.IncrementCount() != 100 {
		t.Errorf("IncrementCount() = %d, want 100", c.IncrementCount())
	}
}

func TestCrashCollectorRecordPanic(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("test error", "/api/users", "GET")

	if c.RecordCount() != 1 {
		t.Errorf("RecordCount() = %d, want 1", c.RecordCount())
	}
}

func TestCrashCollectorLastRecord(t *testing.T) {
	c := NewCrashCollector()

	if c.LastRecord() != nil {
		t.Error("LastRecord() should be nil when empty")
	}

	c.RecordPanic("first", "/first", "GET")
	c.RecordPanic("second", "/second", "POST")

	last := c.LastRecord()
	if last == nil {
		t.Fatal("LastRecord() should not be nil")
	}

	if last.Message != "second" {
		t.Errorf("Message = %v, want %q", last.Message, "second")
	}

	if last.Endpoint != "/second" {
		t.Errorf("Endpoint = %q, want %q", last.Endpoint, "/second")
	}

	if last.Method != "POST" {
		t.Errorf("Method = %q, want %q", last.Method, "POST")
	}
}

func TestCrashCollectorHasRecordFor(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("error", "/api/users", "GET")
	c.RecordPanic("error", "/api/posts", "POST")

	tests := []struct {
		endpoint string
		want     bool
	}{
		{"/api/users", true},
		{"/api/posts", true},
		{"/api/comments", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got := c.HasRecordFor(tt.endpoint)
			if got != tt.want {
				t.Errorf("HasRecordFor(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestCrashCollectorReset(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("error", "/test", "GET")
	c.Reset()

	if c.RecordCount() != 0 {
		t.Errorf("after Reset, RecordCount() = %d, want 0", c.RecordCount())
	}
}

func TestCrashCollectorGetCalls(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("first", "/first", "GET")
	c.RecordPanic("second", "/second", "POST")

	calls := c.GetCalls()
	if len(calls) != 2 {
		t.Fatalf("GetCalls() returned %d calls, want 2", len(calls))
	}

	calls[0].Message = "modified"

	original := c.GetCalls()
	if original[0].Message == "modified" {
		t.Error("GetCalls() should return a copy, not the original slice")
	}
}

func TestCrashCollectorCustomRecordPanicFunc(t *testing.T) {
	var capturedMessage string

	c := NewCrashCollector()
	c.RecordPanicFunc = func(message, endpoint, method string) {
		capturedMessage = message
	}

	c.RecordPanic("test", "/test", "GET")

	if capturedMessage != "test" {
		t.Errorf("capturedMessage = %v, want %q", capturedMessage, "test")
	}
}

func TestCrashCollectorConcurrent(t *testing.T) {
	c := NewCrashCollector()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			c.RecordPanic("error", "/test", "GET")
		}(i)
	}

	wg.Wait()

	if c.RecordCount() != 100 {
		t.Errorf("RecordCount() = %d, want 100", c.RecordCount())
	}
}
