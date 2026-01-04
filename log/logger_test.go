package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{"debug", DebugLevel},
		{"dbg", DebugLevel},
		{"DEBUG", DebugLevel},
		{"DbG", DebugLevel},
		{"info", InfoLevel},
		{"inf", InfoLevel},
		{"INFO", InfoLevel},
		{"InF", InfoLevel},
		{"error", ErrorLevel},
		{"err", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"ErR", ErrorLevel},
		{"unknown", InfoLevel},
		{"", InfoLevel},
		{"warn", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToSlogLevel(t *testing.T) {
	tests := []struct {
		input LogLevel
		want  slog.Level
	}{
		{DebugLevel, slog.LevelDebug},
		{InfoLevel, slog.LevelInfo},
		{ErrorLevel, slog.LevelError},
		{LogLevel(999), slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			got := toSlogLevel(tt.input)
			if got != tt.want {
				t.Errorf("toSlogLevel(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantLevel LogLevel
	}{
		{"debug level", "debug", DebugLevel},
		{"info level", "info", InfoLevel},
		{"error level", "error", ErrorLevel},
		{"default level", "invalid", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			slogLogger, ok := logger.(*slogLogger)
			if !ok {
				t.Fatal("NewLogger did not return *slogLogger")
			}
			if slogLogger.logLevel != tt.wantLevel {
				t.Errorf("logLevel = %v, want %v", slogLogger.logLevel, tt.wantLevel)
			}
		})
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  LogLevel
		method    string
		shouldLog bool
	}{
		{"debug level logs debug", DebugLevel, "Debug", true},
		{"debug level logs info", DebugLevel, "Info", true},
		{"debug level logs error", DebugLevel, "Error", true},
		{"info level skips debug", InfoLevel, "Debug", false},
		{"info level logs info", InfoLevel, "Info", true},
		{"info level logs error", InfoLevel, "Error", true},
		{"error level skips debug", ErrorLevel, "Debug", false},
		{"error level skips info", ErrorLevel, "Info", false},
		{"error level logs error", ErrorLevel, "Error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := newTestLogger(buf, tt.logLevel)

			switch tt.method {
			case "Debug":
				logger.Debug("test message")
			case "Info":
				logger.Info("test message")
			case "Error":
				logger.Error("test message")
			}

			output := buf.String()
			logged := strings.Contains(output, "test message")

			if logged != tt.shouldLog {
				t.Errorf("shouldLog = %v, but logged = %v (output: %q)", tt.shouldLog, logged, output)
			}
		})
	}
}

func TestLoggerFormattedMethods(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  LogLevel
		method    string
		shouldLog bool
	}{
		{"debug level logs debugf", DebugLevel, "Debugf", true},
		{"debug level logs infof", DebugLevel, "Infof", true},
		{"debug level logs errorf", DebugLevel, "Errorf", true},
		{"info level skips debugf", InfoLevel, "Debugf", false},
		{"info level logs infof", InfoLevel, "Infof", true},
		{"info level logs errorf", InfoLevel, "Errorf", true},
		{"error level skips debugf", ErrorLevel, "Debugf", false},
		{"error level skips infof", ErrorLevel, "Infof", false},
		{"error level logs errorf", ErrorLevel, "Errorf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := newTestLogger(buf, tt.logLevel)

			switch tt.method {
			case "Debugf":
				logger.Debugf("formatted %s", "message")
			case "Infof":
				logger.Infof("formatted %s", "message")
			case "Errorf":
				logger.Errorf("formatted %s", "message")
			}

			output := buf.String()
			logged := strings.Contains(output, "formatted message")

			if logged != tt.shouldLog {
				t.Errorf("shouldLog = %v, but logged = %v (output: %q)", tt.shouldLog, logged, output)
			}
		})
	}
}

func TestWith(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf, InfoLevel)

	contextLogger := logger.With("key", "value")

	contextLogger.Info("message")

	output := buf.String()
	if !strings.Contains(output, "message") {
		t.Errorf("expected message in output, got: %q", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("expected context fields in output, got: %q", output)
	}
}

func TestWithPreservesLevel(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  LogLevel
		shouldLog bool
	}{
		{"debug level preserved", DebugLevel, true},
		{"info level preserved", InfoLevel, false},
		{"error level preserved", ErrorLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := newTestLogger(buf, tt.logLevel)

			contextLogger := logger.With("ctx", "value")

			contextLogger.Debug("debug message")

			output := buf.String()
			logged := strings.Contains(output, "debug message")

			if logged != tt.shouldLog {
				t.Errorf("shouldLog = %v, but logged = %v (output: %q)", tt.shouldLog, logged, output)
			}
		})
	}
}

func TestNewNoopLogger(t *testing.T) {
	logger := NewNoopLogger()

	logger.Debug("test")
	logger.Debugf("test %s", "formatted")
	logger.Info("test")
	logger.Infof("test %s", "formatted")
	logger.Error("test")
	logger.Errorf("test %s", "formatted")

	contextLogger := logger.With("key", "value")
	contextLogger.Info("test")
}

func newTestLogger(buf *bytes.Buffer, level LogLevel) *slogLogger {
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: toSlogLevel(level),
	})
	return &slogLogger{
		logger:   slog.New(handler),
		logLevel: level,
	}
}
