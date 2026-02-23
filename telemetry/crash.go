package telemetry

import (
	"runtime"
	"sync"
	"time"
)

// CrashRecorder defines the interface for recording panics.
type CrashRecorder interface {
	RecordPanic(message, endpoint, method string)
}

// CrashEvent represents a recorded panic with metadata.
type CrashEvent struct {
	Type      string   `json:"type"`
	Message   string   `json:"message"`
	Stack     []string `json:"stack"`
	Endpoint  string   `json:"endpoint"`
	Method    string   `json:"method"`
	Count     int64    `json:"count"`
	FirstSeen string   `json:"first_seen"`
	LastSeen  string   `json:"last_seen"`
}

// CrashCollector aggregates and deduplicates panic events.
type CrashCollector struct {
	mu      sync.RWMutex
	crashes map[string]*CrashEvent
}

// NewCrashCollector creates a new CrashCollector.
func NewCrashCollector() *CrashCollector {
	return &CrashCollector{
		crashes: make(map[string]*CrashEvent),
	}
}

// RecordPanic records a panic event with deduplication.
func (c *CrashCollector) RecordPanic(message, endpoint, method string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stack := c.captureStack()
	key := c.crashKey(message, endpoint)
	now := time.Now().UTC().Format(time.RFC3339)

	if existing, ok := c.crashes[key]; ok {
		existing.Count++
		existing.LastSeen = now

		return
	}

	c.crashes[key] = &CrashEvent{
		Type:      "panic",
		Message:   c.sanitizeMessage(message),
		Stack:     stack,
		Endpoint:  endpoint,
		Method:    method,
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
}

// GetAndResetCrashes returns all crashes and clears the collector.
func (c *CrashCollector) GetAndResetCrashes() []CrashEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.crashes) == 0 {
		return nil
	}

	result := make([]CrashEvent, 0, len(c.crashes))
	for _, crash := range c.crashes {
		result = append(result, *crash)
	}

	c.crashes = make(map[string]*CrashEvent)

	return result
}

func (c *CrashCollector) crashKey(message, endpoint string) string {
	return message + "|" + endpoint
}

func (c *CrashCollector) captureStack() []string {
	var stack []string

	pcs := make([]uintptr, 20)
	n := runtime.Callers(4, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()

		funcName := frame.Function
		if funcName != "" {
			stack = append(stack, funcName)
		}

		if !more || len(stack) >= 10 {
			break
		}
	}

	return stack
}

func (c *CrashCollector) sanitizeMessage(msg string) string {
	if len(msg) > 200 {
		return msg[:200]
	}

	return msg
}
