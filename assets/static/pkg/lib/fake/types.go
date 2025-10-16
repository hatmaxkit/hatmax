package fake

// Common types used across fakes to maintain consistency

// Logger interface that fakes can implement
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