package middleware

import (
	"fmt"
	"net/http"
)

// TelemetryCounter returns middleware that increments a counter on each request.
func TelemetryCounter(counter RequestCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if counter != nil {
				counter.IncrementRequests()
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TelemetryRecovery returns middleware that recovers from panics and records them.
func TelemetryRecovery(recorder CrashRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if recorder != nil {
						recorder.RecordPanic(fmt.Sprintf("%v", rec), r.URL.Path, r.Method)
					}
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
