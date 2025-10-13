package core

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	ErrorLevel
)

type Logger interface {
	Debug(v ...any)
	Debugf(format string, a ...any)
	Info(v ...any)
	Infof(format string, a ...any)
	Error(v ...any)
	Errorf(format string, a ...any)
	SetLogLevel(level LogLevel)
	With(args ...any) Logger
}

type slogLogger struct {
	logger   *slog.Logger
	logLevel LogLevel
}

func NewLogger(logLevelStr string) Logger {
	level := toValidLevel(logLevelStr)

	var handler slog.Handler
	if isTerminal() {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slogLevel(level),
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slogLevel(level),
		})
	}

	return &slogLogger{
		logger:   slog.New(handler),
		logLevel: level,
	}
}

func (l *slogLogger) Debug(v ...any) {
	if l.logLevel <= DebugLevel {
		l.logger.Debug(fmt.Sprint(v...))
	}
}

func (l *slogLogger) Debugf(format string, a ...any) {
	if l.logLevel <= DebugLevel {
		l.logger.Debug(fmt.Sprintf(format, a...))
	}
}

func (l *slogLogger) Info(v ...any) {
	if l.logLevel <= InfoLevel {
		l.logger.Info(fmt.Sprint(v...))
	}
}

func (l *slogLogger) Infof(format string, a ...any) {
	if l.logLevel <= InfoLevel {
		l.logger.Info(fmt.Sprintf(format, a...))
	}
}

func (l *slogLogger) Error(v ...any) {
	if l.logLevel <= ErrorLevel {
		l.logger.Error(fmt.Sprint(v...))
	}
}

func (l *slogLogger) Errorf(format string, a ...any) {
	if l.logLevel <= ErrorLevel {
		l.logger.Error(fmt.Sprintf(format, a...))
	}
}

func (l *slogLogger) SetLogLevel(level LogLevel) {
	l.logLevel = level
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger:   l.logger.With(args...),
		logLevel: l.logLevel,
	}
}

type noopLogger struct{}

func (noopLogger) Debug(v ...any)                 {}
func (noopLogger) Debugf(format string, a ...any) {}
func (noopLogger) Info(v ...any)                  {}
func (noopLogger) Infof(format string, a ...any)  {}
func (noopLogger) Error(v ...any)                 {}
func (noopLogger) Errorf(format string, a ...any) {}
func (noopLogger) SetLogLevel(level LogLevel)     {}
func (noopLogger) With(args ...any) Logger        { return noopLogger{} }

func NewNoopLogger() Logger {
	return noopLogger{}
}

func toValidLevel(level string) LogLevel {
	level = strings.ToLower(level)
	switch level {
	case "debug", "dbg":
		return DebugLevel
	case "info", "inf":
		return InfoLevel
	case "error", "err":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

func slogLevel(level LogLevel) slog.Level {
	switch level {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func isTerminal() bool {
	return os.Getenv("LOG_FORMAT") != "json"
}
