package fake

import (
	"fmt"
	"strings"
	"sync"
)

// LogLevel represents log levels
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	ErrorLevel
)

// Logger fake implementation with programmable responses and call tracking
type Logger struct {
	mu sync.RWMutex

	// Configuration
	logLevel LogLevel
	enabled  bool

	// Call tracking
	DebugCalls  []LogCall
	DebugfCalls []LogfCall
	InfoCalls   []LogCall
	InfofCalls  []LogfCall
	ErrorCalls  []LogCall
	ErrorfCalls []LogfCall
	WithCalls   []WithCall
	SetLogLevelCalls []SetLogLevelCall

	// Captured output
	Messages []LogMessage
}

// Call structures for tracking
type LogCall struct {
	Args []any
}

type LogfCall struct {
	Format string
	Args   []any
}

type WithCall struct {
	Args []any
}

type SetLogLevelCall struct {
	Level LogLevel
}

type LogMessage struct {
	Level   LogLevel
	Message string
	Args    []any
}

// NewLogger creates a new fake Logger
func NewLogger() *Logger {
	return &Logger{
		enabled:  true,
		logLevel: InfoLevel,
	}
}

// Reset clears all call history and messages
func (f *Logger) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logLevel = InfoLevel
	f.enabled = true
	f.DebugCalls = nil
	f.DebugfCalls = nil
	f.InfoCalls = nil
	f.InfofCalls = nil
	f.ErrorCalls = nil
	f.ErrorfCalls = nil
	f.WithCalls = nil
	f.SetLogLevelCalls = nil
	f.Messages = nil
}

// Debug implements Logger interface
func (f *Logger) Debug(v ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.DebugCalls = append(f.DebugCalls, LogCall{Args: v})

	// Capture message if enabled and level allows
	if f.enabled && f.logLevel <= DebugLevel {
		msg := fmt.Sprint(v...)
		f.Messages = append(f.Messages, LogMessage{
			Level:   DebugLevel,
			Message: msg,
			Args:    v,
		})
	}
}

// Debugf implements Logger interface
func (f *Logger) Debugf(format string, a ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.DebugfCalls = append(f.DebugfCalls, LogfCall{Format: format, Args: a})

	// Capture message if enabled and level allows
	if f.enabled && f.logLevel <= DebugLevel {
		msg := fmt.Sprintf(format, a...)
		f.Messages = append(f.Messages, LogMessage{
			Level:   DebugLevel,
			Message: msg,
			Args:    a,
		})
	}
}

// Info implements Logger interface
func (f *Logger) Info(v ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.InfoCalls = append(f.InfoCalls, LogCall{Args: v})

	// Capture message if enabled and level allows
	if f.enabled && f.logLevel <= InfoLevel {
		msg := fmt.Sprint(v...)
		f.Messages = append(f.Messages, LogMessage{
			Level:   InfoLevel,
			Message: msg,
			Args:    v,
		})
	}
}

// Infof implements Logger interface
func (f *Logger) Infof(format string, a ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.InfofCalls = append(f.InfofCalls, LogfCall{Format: format, Args: a})

	// Capture message if enabled and level allows
	if f.enabled && f.logLevel <= InfoLevel {
		msg := fmt.Sprintf(format, a...)
		f.Messages = append(f.Messages, LogMessage{
			Level:   InfoLevel,
			Message: msg,
			Args:    a,
		})
	}
}

// Error implements Logger interface
func (f *Logger) Error(v ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.ErrorCalls = append(f.ErrorCalls, LogCall{Args: v})

	// Capture message if enabled and level allows
	if f.enabled && f.logLevel <= ErrorLevel {
		msg := fmt.Sprint(v...)
		f.Messages = append(f.Messages, LogMessage{
			Level:   ErrorLevel,
			Message: msg,
			Args:    v,
		})
	}
}

// Errorf implements Logger interface
func (f *Logger) Errorf(format string, a ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.ErrorfCalls = append(f.ErrorfCalls, LogfCall{Format: format, Args: a})

	// Capture message if enabled and level allows
	if f.enabled && f.logLevel <= ErrorLevel {
		msg := fmt.Sprintf(format, a...)
		f.Messages = append(f.Messages, LogMessage{
			Level:   ErrorLevel,
			Message: msg,
			Args:    a,
		})
	}
}

// SetLogLevel implements Logger interface
func (f *Logger) SetLogLevel(level LogLevel) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.SetLogLevelCalls = append(f.SetLogLevelCalls, SetLogLevelCall{Level: level})

	// Apply the level change
	f.logLevel = level
}

// With implements Logger interface
func (f *Logger) With(args ...any) Logger {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.WithCalls = append(f.WithCalls, WithCall{Args: args})

	// Return a new logger that shares the same call tracking
	// In a real implementation, this would create a child logger with context
	// For testing purposes, we return self to keep tracking centralized
	return f
}

// Helper methods for testing

// SetEnabled controls whether messages are captured
func (f *Logger) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

// CallCount returns the number of calls to a specific method
func (f *Logger) CallCount(method string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	switch method {
	case "Debug":
		return len(f.DebugCalls)
	case "Debugf":
		return len(f.DebugfCalls)
	case "Info":
		return len(f.InfoCalls)
	case "Infof":
		return len(f.InfofCalls)
	case "Error":
		return len(f.ErrorCalls)
	case "Errorf":
		return len(f.ErrorfCalls)
	case "With":
		return len(f.WithCalls)
	case "SetLogLevel":
		return len(f.SetLogLevelCalls)
	default:
		return 0
	}
}

// GetMessages returns all captured messages
func (f *Logger) GetMessages() []LogMessage {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	// Return copy to avoid race conditions
	messages := make([]LogMessage, len(f.Messages))
	copy(messages, f.Messages)
	return messages
}

// GetMessagesByLevel returns messages filtered by level
func (f *Logger) GetMessagesByLevel(level LogLevel) []LogMessage {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	var filtered []LogMessage
	for _, msg := range f.Messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// HasMessage checks if a message containing the text was logged
func (f *Logger) HasMessage(text string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	for _, msg := range f.Messages {
		if strings.Contains(msg.Message, text) {
			return true
		}
	}
	return false
}

// HasMessageAtLevel checks if a message containing the text was logged at the specified level
func (f *Logger) HasMessageAtLevel(level LogLevel, text string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	for _, msg := range f.Messages {
		if msg.Level == level && strings.Contains(msg.Message, text) {
			return true
		}
	}
	return false
}

// GetCurrentLevel returns the current log level
func (f *Logger) GetCurrentLevel() LogLevel {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.logLevel
}