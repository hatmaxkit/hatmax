package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticCache(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		wantCacheCtrl string
	}{
		{
			name:          "GET request sets cache headers",
			method:        http.MethodGet,
			wantCacheCtrl: "public, max-age=31536000, immutable",
		},
		{
			name:          "HEAD request sets cache headers",
			method:        http.MethodHead,
			wantCacheCtrl: "public, max-age=31536000, immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := StaticCache(inner)

			req := httptest.NewRequest(tt.method, "/static/test.css", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			got := rec.Header().Get("Cache-Control")
			if got != tt.wantCacheCtrl {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheCtrl)
			}
		})
	}
}

func TestStaticCacheCallsNext(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		w.WriteHeader(http.StatusOK)
	})

	handler := StaticCache(inner)

	req := httptest.NewRequest(http.MethodGet, "/static/test.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
