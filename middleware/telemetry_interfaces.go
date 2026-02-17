package middleware

// RequestCounter increments on each HTTP request.
type RequestCounter interface {
	IncrementRequests()
}

// CrashRecorder records panic information.
type CrashRecorder interface {
	RecordPanic(message, endpoint, method string)
}
