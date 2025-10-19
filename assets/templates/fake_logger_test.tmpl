package fake

import (
	"sync"
	"testing"
)

func TestLoggerBasicFunctionality(t *testing.T) {
	logger := NewLogger()

	// Test Debug
	logger.Debug("debug message", "key", "value")
	if logger.CallCount("Debug") != 1 {
		t.Errorf("Expected 1 Debug call, got %d", logger.CallCount("Debug"))
	}

	// Test Info
	logger.Info("info message")
	if logger.CallCount("Info") != 1 {
		t.Errorf("Expected 1 Info call, got %d", logger.CallCount("Info"))
	}

	// Test Error
	logger.Error("error message")
	if logger.CallCount("Error") != 1 {
		t.Errorf("Expected 1 Error call, got %d", logger.CallCount("Error"))
	}

	// Check messages were captured (Info and Error should be captured, Debug should not at InfoLevel)
	messages := logger.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestLoggerFormattedMethods(t *testing.T) {
	logger := NewLogger()
	logger.SetLogLevel(DebugLevel) // Enable debug logging

	// Test formatted methods
	logger.Debugf("debug: %s = %d", "count", 42)
	logger.Infof("info: %s", "test")
	logger.Errorf("error: %v", "failure")

	// Check call counts
	if logger.CallCount("Debugf") != 1 {
		t.Errorf("Expected 1 Debugf call, got %d", logger.CallCount("Debugf"))
	}
	if logger.CallCount("Infof") != 1 {
		t.Errorf("Expected 1 Infof call, got %d", logger.CallCount("Infof"))
	}
	if logger.CallCount("Errorf") != 1 {
		t.Errorf("Expected 1 Errorf call, got %d", logger.CallCount("Errorf"))
	}

	// Check formatted messages
	messages := logger.GetMessages()
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Verify message formatting
	if !logger.HasMessage("debug: count = 42") {
		t.Error("Expected formatted debug message")
	}
	if !logger.HasMessage("info: test") {
		t.Error("Expected formatted info message")
	}
	if !logger.HasMessage("error: failure") {
		t.Error("Expected formatted error message")
	}
}

func TestLoggerLevels(t *testing.T) {
	logger := NewLogger()

	tests := []struct {
		name     string
		level    LogLevel
		expected map[LogLevel]bool
	}{
		{
			name:  "DebugLevel allows all",
			level: DebugLevel,
			expected: map[LogLevel]bool{
				DebugLevel: true,
				InfoLevel:  true,
				ErrorLevel: true,
			},
		},
		{
			name:  "InfoLevel blocks debug",
			level: InfoLevel,
			expected: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  true,
				ErrorLevel: true,
			},
		},
		{
			name:  "ErrorLevel blocks debug and info",
			level: ErrorLevel,
			expected: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  false,
				ErrorLevel: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.Reset()
			logger.SetLogLevel(tt.level)

			// Log at each level
			logger.Debug("debug")
			logger.Info("info")
			logger.Error("error")

			// Check what was captured
			debugMessages := logger.GetMessagesByLevel(DebugLevel)
			infoMessages := logger.GetMessagesByLevel(InfoLevel)
			errorMessages := logger.GetMessagesByLevel(ErrorLevel)

			if tt.expected[DebugLevel] && len(debugMessages) == 0 {
				t.Error("Expected debug messages to be captured")
			}
			if !tt.expected[DebugLevel] && len(debugMessages) > 0 {
				t.Error("Expected debug messages to be filtered out")
			}

			if tt.expected[InfoLevel] && len(infoMessages) == 0 {
				t.Error("Expected info messages to be captured")
			}
			if !tt.expected[InfoLevel] && len(infoMessages) > 0 {
				t.Error("Expected info messages to be filtered out")
			}

			if tt.expected[ErrorLevel] && len(errorMessages) == 0 {
				t.Error("Expected error messages to be captured")
			}
			if !tt.expected[ErrorLevel] && len(errorMessages) > 0 {
				t.Error("Expected error messages to be filtered out")
			}
		})
	}
}

func TestLoggerSetLogLevel(t *testing.T) {
	logger := NewLogger()

	// Initial level should be InfoLevel
	if logger.GetCurrentLevel() != InfoLevel {
		t.Errorf("Expected initial level InfoLevel, got %v", logger.GetCurrentLevel())
	}

	// Change to DebugLevel
	logger.SetLogLevel(DebugLevel)
	if logger.GetCurrentLevel() != DebugLevel {
		t.Errorf("Expected DebugLevel, got %v", logger.GetCurrentLevel())
	}

	// Check call was tracked
	if logger.CallCount("SetLogLevel") != 1 {
		t.Errorf("Expected 1 SetLogLevel call, got %d", logger.CallCount("SetLogLevel"))
	}

	// Change to ErrorLevel
	logger.SetLogLevel(ErrorLevel)
	if logger.GetCurrentLevel() != ErrorLevel {
		t.Errorf("Expected ErrorLevel, got %v", logger.GetCurrentLevel())
	}

	if logger.CallCount("SetLogLevel") != 2 {
		t.Errorf("Expected 2 SetLogLevel calls, got %d", logger.CallCount("SetLogLevel"))
	}
}

func TestLoggerWith(t *testing.T) {
	logger := NewLogger()

	// Test With method
	childLogger := logger.With("key", "value", "count", 42)

	// Check call was tracked
	if logger.CallCount("With") != 1 {
		t.Errorf("Expected 1 With call, got %d", logger.CallCount("With"))
	}

	// In our fake implementation, With returns self for simplicity
	// This allows centralized call tracking
	if childLogger != logger {
		t.Error("Expected With to return same logger for testing")
	}

	// Test that child logger logs are tracked in parent
	childLogger.Info("child message")
	if logger.CallCount("Info") != 1 {
		t.Errorf("Expected child Info call to be tracked, got %d", logger.CallCount("Info"))
	}
}

func TestLoggerEnabledDisabled(t *testing.T) {
	logger := NewLogger()
	logger.SetLogLevel(DebugLevel)

	// Log when enabled (default)
	logger.Debug("enabled debug")
	logger.Info("enabled info")

	// Disable logging
	logger.SetEnabled(false)
	logger.Debug("disabled debug")
	logger.Info("disabled info")

	// Re-enable logging
	logger.SetEnabled(true)
	logger.Debug("re-enabled debug")

	// Check messages - only enabled messages should be captured
	messages := logger.GetMessages()
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Check specific messages
	if !logger.HasMessage("enabled debug") {
		t.Error("Expected 'enabled debug' message")
	}
	if !logger.HasMessage("enabled info") {
		t.Error("Expected 'enabled info' message")
	}
	if !logger.HasMessage("re-enabled debug") {
		t.Error("Expected 're-enabled debug' message")
	}
	if logger.HasMessage("disabled debug") {
		t.Error("Should not have 'disabled debug' message")
	}
	if logger.HasMessage("disabled info") {
		t.Error("Should not have 'disabled info' message")
	}

	// But calls should still be tracked even when disabled
	if logger.CallCount("Debug") != 3 {
		t.Errorf("Expected 3 Debug calls, got %d", logger.CallCount("Debug"))
	}
	if logger.CallCount("Info") != 2 {
		t.Errorf("Expected 2 Info calls, got %d", logger.CallCount("Info"))
	}
}

func TestLoggerHasMessage(t *testing.T) {
	logger := NewLogger()
	logger.SetLogLevel(DebugLevel)

	// Log some messages
	logger.Debug("debug: processing item 123")
	logger.Info("info: operation completed successfully")
	logger.Error("error: error connect to database")

	// Test HasMessage
	if !logger.HasMessage("processing item 123") {
		t.Error("Expected to find debug message")
	}
	if !logger.HasMessage("operation completed") {
		t.Error("Expected to find info message")
	}
	if !logger.HasMessage("error connect") {
		t.Error("Expected to find error message")
	}
	if logger.HasMessage("nonexistent message") {
		t.Error("Should not find nonexistent message")
	}

	// Test HasMessageAtLevel
	if !logger.HasMessageAtLevel(DebugLevel, "processing item") {
		t.Error("Expected to find debug message at debug level")
	}
	if !logger.HasMessageAtLevel(InfoLevel, "operation completed") {
		t.Error("Expected to find info message at info level")
	}
	if !logger.HasMessageAtLevel(ErrorLevel, "error connect") {
		t.Error("Expected to find error message at error level")
	}

	// Test wrong level
	if logger.HasMessageAtLevel(ErrorLevel, "processing item") {
		t.Error("Should not find debug message at error level")
	}
	if logger.HasMessageAtLevel(DebugLevel, "error connect") {
		t.Error("Should not find error message at debug level")
	}
}

func TestLoggerReset(t *testing.T) {
	logger := NewLogger()

	// Add some data
	logger.SetLogLevel(DebugLevel)
	logger.SetEnabled(false)
	logger.Debug("test")
	logger.Info("test")
	logger.With("key", "value")

	// Verify data exists
	if logger.GetCurrentLevel() == InfoLevel {
		t.Error("Expected level to be changed before reset")
	}
	if logger.CallCount("Debug") == 0 {
		t.Error("Expected calls to exist before reset")
	}

	// Reset
	logger.Reset()

	// Verify everything is cleared
	if logger.GetCurrentLevel() != InfoLevel {
		t.Error("Expected level to be reset to InfoLevel")
	}
	if logger.CallCount("Debug") != 0 {
		t.Error("Expected call history to be cleared after reset")
	}
	if len(logger.GetMessages()) != 0 {
		t.Error("Expected messages to be cleared after reset")
	}
	if logger.CallCount("With") != 0 {
		t.Error("Expected With calls to be cleared after reset")
	}
	if logger.CallCount("SetLogLevel") != 0 {
		t.Error("Expected SetLogLevel calls to be cleared after reset")
	}
}

func TestLoggerConcurrentAccess(t *testing.T) {
	logger := NewLogger()
	logger.SetLogLevel(DebugLevel)

	const numGoroutines = 10
	const numOperationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Test concurrent logging
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				logger.Debug("debug", goroutineID, j)
				logger.Info("info", goroutineID, j)
				logger.Error("error", goroutineID, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify all calls were tracked
	expectedCalls := numGoroutines * numOperationsPerGoroutine
	if logger.CallCount("Debug") != expectedCalls {
		t.Errorf("Expected %d Debug calls, got %d", expectedCalls, logger.CallCount("Debug"))
	}
	if logger.CallCount("Info") != expectedCalls {
		t.Errorf("Expected %d Info calls, got %d", expectedCalls, logger.CallCount("Info"))
	}
	if logger.CallCount("Error") != expectedCalls {
		t.Errorf("Expected %d Error calls, got %d", expectedCalls, logger.CallCount("Error"))
	}

	// Verify all messages were captured
	messages := logger.GetMessages()
	expectedMessages := expectedCalls * 3 // Debug + Info + Error
	if len(messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(messages))
	}

	// Test concurrent With calls don't panic
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				logger.With("goroutine", goroutineID, "operation", j)
			}
		}(i)
	}

	wg.Wait()

	// Verify With calls were tracked
	if logger.CallCount("With") != expectedCalls {
		t.Errorf("Expected %d With calls, got %d", expectedCalls, logger.CallCount("With"))
	}
}

func TestLoggerUnknownMethod(t *testing.T) {
	logger := NewLogger()

	// Test unknown method
	if count := logger.CallCount("UnknownMethod"); count != 0 {
		t.Errorf("Expected 0 for unknown method, got %d", count)
	}
}