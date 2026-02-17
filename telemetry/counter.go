package telemetry

import "sync/atomic"

// RequestCounter defines the interface for counting HTTP requests.
type RequestCounter interface {
	IncrementRequests()
	GetAndResetRequests() int64
}

// AtomicCounter provides a thread-safe request counter.
type AtomicCounter struct {
	count atomic.Int64
}

// NewCounter creates a new AtomicCounter.
func NewCounter() *AtomicCounter {
	return &AtomicCounter{}
}

// IncrementRequests increments the request count by one.
func (c *AtomicCounter) IncrementRequests() {
	c.count.Add(1)
}

// GetAndResetRequests returns the current count and resets it to zero.
func (c *AtomicCounter) GetAndResetRequests() int64 {
	return c.count.Swap(0)
}
