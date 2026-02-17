package fake

import "sync"

// CounterCall records an IncrementRequests call.
type CounterCall struct{}

// Counter is a test double for telemetry.RequestCounter.
type Counter struct {
	mu sync.Mutex

	IncrementFunc   func()
	incrementCalls  []CounterCall
	GetAndResetFunc func() int64
	getAndResetCalls int
	simulatedCount  int64
}

// NewCounter creates a new fake Counter.
func NewCounter() *Counter {
	return &Counter{}
}

// IncrementRequests records the call and invokes IncrementFunc if set.
func (c *Counter) IncrementRequests() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.incrementCalls = append(c.incrementCalls, CounterCall{})
	c.simulatedCount++

	if c.IncrementFunc != nil {
		c.IncrementFunc()
	}
}

// GetAndResetRequests returns simulatedCount and resets it, or calls GetAndResetFunc if set.
func (c *Counter) GetAndResetRequests() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.getAndResetCalls++

	if c.GetAndResetFunc != nil {
		return c.GetAndResetFunc()
	}

	count := c.simulatedCount
	c.simulatedCount = 0
	return count
}

// Reset clears all recorded calls and the simulated count.
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.incrementCalls = nil
	c.getAndResetCalls = 0
	c.simulatedCount = 0
}

// IncrementCount returns the number of IncrementRequests calls.
func (c *Counter) IncrementCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.incrementCalls)
}

// GetAndResetCount returns the number of GetAndResetRequests calls.
func (c *Counter) GetAndResetCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getAndResetCalls
}

// WithCount sets the simulated count and returns the Counter for chaining.
func (c *Counter) WithCount(n int64) *Counter {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.simulatedCount = n
	return c
}

// CrashCall records a RecordPanic call.
type CrashCall struct {
	Message  string
	Endpoint string
	Method   string
}

// CrashCollector is a test double for telemetry.CrashRecorder.
type CrashCollector struct {
	mu sync.Mutex

	RecordPanicFunc func(message, endpoint, method string)
	calls           []CrashCall
}

// NewCrashCollector creates a new fake CrashCollector.
func NewCrashCollector() *CrashCollector {
	return &CrashCollector{}
}

// RecordPanic records the call and invokes RecordPanicFunc if set.
func (c *CrashCollector) RecordPanic(message, endpoint, method string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls = append(c.calls, CrashCall{
		Message:  message,
		Endpoint: endpoint,
		Method:   method,
	})

	if c.RecordPanicFunc != nil {
		c.RecordPanicFunc(message, endpoint, method)
	}
}

// Reset clears all recorded calls.
func (c *CrashCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = nil
}

// RecordCount returns the number of RecordPanic calls.
func (c *CrashCollector) RecordCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

// LastRecord returns the most recent CrashCall or nil if none.
func (c *CrashCollector) LastRecord() *CrashCall {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.calls) == 0 {
		return nil
	}
	call := c.calls[len(c.calls)-1]
	return &call
}

// HasRecordFor returns true if any call was recorded for the given endpoint.
func (c *CrashCollector) HasRecordFor(endpoint string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, call := range c.calls {
		if call.Endpoint == endpoint {
			return true
		}
	}
	return false
}

// GetCalls returns a copy of all recorded calls.
func (c *CrashCollector) GetCalls() []CrashCall {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]CrashCall, len(c.calls))
	copy(result, c.calls)
	return result
}
