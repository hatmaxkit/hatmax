package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type fakeCounter struct {
	mu    sync.Mutex
	count int
}

func (f *fakeCounter) IncrementRequests() {
	f.mu.Lock()
	f.count++
	f.mu.Unlock()
}

func (f *fakeCounter) getCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.count
}

type fakeCrashRecorder struct {
	mu       sync.Mutex
	calls    []crashCall
	disabled bool
}

type crashCall struct {
	message  string
	endpoint string
	method   string
}

func (f *fakeCrashRecorder) RecordPanic(message, endpoint, method string) {
	f.mu.Lock()
	f.calls = append(f.calls, crashCall{message, endpoint, method})
	f.mu.Unlock()
}

func (f *fakeCrashRecorder) getCalls() []crashCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]crashCall, len(f.calls))
	copy(result, f.calls)

	return result
}

func TestTelemetryCounter(t *testing.T) {
	t.Run("single request increments counter", func(t *testing.T) {
		counter := &fakeCounter{}
		handlerCalled := 0
		handler := TelemetryCounter(counter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled++

			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		if counter.getCount() != 1 {
			t.Errorf("counter = %d, want 1", counter.getCount())
		}

		if handlerCalled != 1 {
			t.Errorf("handler called %d times, want 1", handlerCalled)
		}
	})

	t.Run("multiple requests increment counter", func(t *testing.T) {
		counter := &fakeCounter{}
		handler := TelemetryCounter(counter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for range 5 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		if counter.getCount() != 5 {
			t.Errorf("counter = %d, want 5", counter.getCount())
		}
	})

	t.Run("nil counter does not panic", func(t *testing.T) {
		handlerCalled := 0
		handler := TelemetryCounter(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled++

			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		if handlerCalled != 1 {
			t.Errorf("handler called %d times, want 1", handlerCalled)
		}
	})
}

func TestTelemetryRecovery(t *testing.T) {
	t.Run("no panic passes through", func(t *testing.T) {
		recorder := &fakeCrashRecorder{}
		handler := TelemetryRecovery(recorder)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		if len(recorder.getCalls()) != 0 {
			t.Errorf("records = %d, want 0", len(recorder.getCalls()))
		}
	})

	t.Run("panic is caught and recorded", func(t *testing.T) {
		recorder := &fakeCrashRecorder{}
		handler := TelemetryRecovery(recorder)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}

		calls := recorder.getCalls()
		if len(calls) != 1 {
			t.Fatalf("records = %d, want 1", len(calls))
		}

		if calls[0].message != "test panic" {
			t.Errorf("message = %v, want %q", calls[0].message, "test panic")
		}

		if calls[0].endpoint != "/api/test" {
			t.Errorf("endpoint = %q, want %q", calls[0].endpoint, "/api/test")
		}

		if calls[0].method != http.MethodPost {
			t.Errorf("method = %q, want %q", calls[0].method, http.MethodPost)
		}
	})

	t.Run("panic with nil recorder does not panic", func(t *testing.T) {
		handler := TelemetryRecovery(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestTelemetryRecoveryErrorMessage(t *testing.T) {
	recorder := &fakeCrashRecorder{}

	handler := TelemetryRecovery(recorder)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if body != "Internal Server Error\n" {
		t.Errorf("body = %q, want %q", body, "Internal Server Error\n")
	}
}
