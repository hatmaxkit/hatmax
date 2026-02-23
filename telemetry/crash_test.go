package telemetry

import (
	"strings"
	"sync"
	"testing"
)

var _ interface {
	RecordPanic(message, endpoint, method string)
} = (*CrashCollector)(nil)

func TestCrashCollectorRecordSingle(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("test error", "/api/users", "GET")

	crashes := c.GetAndResetCrashes()
	if len(crashes) != 1 {
		t.Fatalf("got %d crashes, want 1", len(crashes))
	}

	crash := crashes[0]
	if crash.Type != "panic" {
		t.Errorf("Type = %q, want %q", crash.Type, "panic")
	}

	if crash.Message != "test error" {
		t.Errorf("Message = %q, want %q", crash.Message, "test error")
	}

	if crash.Endpoint != "/api/users" {
		t.Errorf("Endpoint = %q, want %q", crash.Endpoint, "/api/users")
	}

	if crash.Method != "GET" {
		t.Errorf("Method = %q, want %q", crash.Method, "GET")
	}

	if crash.Count != 1 {
		t.Errorf("Count = %d, want 1", crash.Count)
	}

	if crash.FirstSeen == "" {
		t.Error("FirstSeen should not be empty")
	}

	if crash.LastSeen == "" {
		t.Error("LastSeen should not be empty")
	}

	if len(crash.Stack) == 0 {
		t.Error("Stack should not be empty")
	}
}

func TestCrashCollectorDeduplication(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("same error", "/api/users", "GET")
	c.RecordPanic("same error", "/api/users", "GET")
	c.RecordPanic("same error", "/api/users", "GET")

	crashes := c.GetAndResetCrashes()
	if len(crashes) != 1 {
		t.Fatalf("got %d crashes, want 1 (deduplicated by message+endpoint)", len(crashes))
	}

	crash := crashes[0]
	if crash.Count != 3 {
		t.Errorf("Count = %d, want 3", crash.Count)
	}

	if crash.Endpoint != "/api/users" {
		t.Errorf("Endpoint = %q, want %q", crash.Endpoint, "/api/users")
	}
}

func TestCrashCollectorDifferentEndpoints(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("error", "/api/users", "GET")
	c.RecordPanic("error", "/api/posts", "GET")
	c.RecordPanic("error", "/api/comments", "GET")

	crashes := c.GetAndResetCrashes()
	if len(crashes) != 3 {
		t.Errorf("got %d crashes, want 3", len(crashes))
	}
}

func TestCrashCollectorReset(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("error", "/test", "GET")

	first := c.GetAndResetCrashes()
	if len(first) != 1 {
		t.Errorf("first call got %d crashes, want 1", len(first))
	}

	second := c.GetAndResetCrashes()
	if second != nil {
		t.Errorf("second call got %v, want nil", second)
	}
}

func TestCrashCollectorEmptyReturnsNil(t *testing.T) {
	c := NewCrashCollector()

	crashes := c.GetAndResetCrashes()
	if crashes != nil {
		t.Errorf("got %v, want nil", crashes)
	}
}

func TestCrashCollectorSanitizeMessage(t *testing.T) {
	c := NewCrashCollector()

	longMessage := strings.Repeat("x", 300)
	c.RecordPanic(longMessage, "/test", "GET")

	crashes := c.GetAndResetCrashes()
	if len(crashes) != 1 {
		t.Fatalf("got %d crashes, want 1", len(crashes))
	}

	if len(crashes[0].Message) != 200 {
		t.Errorf("Message length = %d, want 200", len(crashes[0].Message))
	}
}

func TestCrashCollectorStackCapture(t *testing.T) {
	c := NewCrashCollector()

	c.RecordPanic("error", "/test", "GET")

	crashes := c.GetAndResetCrashes()
	if len(crashes) != 1 {
		t.Fatalf("got %d crashes, want 1", len(crashes))
	}

	if len(crashes[0].Stack) == 0 {
		t.Error("Stack should contain frames")
	}

	if len(crashes[0].Stack) > 10 {
		t.Errorf("Stack has %d frames, want <= 10", len(crashes[0].Stack))
	}
}

func TestCrashCollectorConcurrent(t *testing.T) {
	c := NewCrashCollector()

	const (
		goroutines         = 50
		panicsPerGoroutine = 20
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			for range panicsPerGoroutine {
				c.RecordPanic("concurrent error", "/test", "GET")
			}
		}()
	}

	wg.Wait()

	crashes := c.GetAndResetCrashes()
	if len(crashes) != 1 {
		t.Fatalf("got %d crashes, want 1 (all deduplicated)", len(crashes))
	}

	totalCount := int64(goroutines * panicsPerGoroutine)
	if crashes[0].Count != totalCount {
		t.Errorf("Count = %d, want %d", crashes[0].Count, totalCount)
	}
}
