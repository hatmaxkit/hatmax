package telemetry

import (
	"sync"
	"testing"
)

var _ RequestCounter = (*AtomicCounter)(nil)

func TestAtomicCounter(t *testing.T) {
	tests := []struct {
		name       string
		increments int
		wantCount  int64
	}{
		{
			name:       "initial zero value",
			increments: 0,
			wantCount:  0,
		},
		{
			name:       "single increment",
			increments: 1,
			wantCount:  1,
		},
		{
			name:       "multiple increments",
			increments: 100,
			wantCount:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCounter()

			for i := 0; i < tt.increments; i++ {
				c.IncrementRequests()
			}

			got := c.GetAndResetRequests()
			if got != tt.wantCount {
				t.Errorf("GetAndResetRequests() = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestAtomicCounterReset(t *testing.T) {
	c := NewCounter()

	c.IncrementRequests()
	c.IncrementRequests()
	c.IncrementRequests()

	first := c.GetAndResetRequests()
	if first != 3 {
		t.Errorf("first GetAndResetRequests() = %d, want 3", first)
	}

	second := c.GetAndResetRequests()
	if second != 0 {
		t.Errorf("second GetAndResetRequests() = %d, want 0", second)
	}

	c.IncrementRequests()
	third := c.GetAndResetRequests()
	if third != 1 {
		t.Errorf("third GetAndResetRequests() = %d, want 1", third)
	}
}

func TestAtomicCounterConcurrent(t *testing.T) {
	c := NewCounter()

	const goroutines = 100
	const incrementsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range incrementsPerGoroutine {
				c.IncrementRequests()
			}
		}()
	}

	wg.Wait()

	got := c.GetAndResetRequests()
	want := int64(goroutines * incrementsPerGoroutine)
	if got != want {
		t.Errorf("concurrent increments = %d, want %d", got, want)
	}
}
