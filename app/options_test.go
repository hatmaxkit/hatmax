package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/log"
)

func TestWithPing(t *testing.T) {
	r := chi.NewRouter()

	if err := ApplyRouterOptions(r, WithPing()); err != nil {
		t.Fatalf("ApplyRouterOptions() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handlePing() status = %d, want %d", rec.Code, http.StatusOK)
	}

	want := `{"status":"ok"}`
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("handlePing() body = %q, want to contain %q", rec.Body.String(), want)
	}
}

func TestWithDebugRoutes(t *testing.T) {
	r := chi.NewRouter()

	// Add some test routes
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {})
	r.Post("/users", func(w http.ResponseWriter, r *http.Request) {})

	if err := ApplyRouterOptions(r, WithDebugRoutes()); err != nil {
		t.Fatalf("ApplyRouterOptions() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/debug/routes", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleDebugRoutes() status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Registered Routes:") {
		t.Error("handleDebugRoutes() body should contain 'Registered Routes:'")
	}

	if !strings.Contains(body, "/test") {
		t.Error("handleDebugRoutes() body should contain '/test'")
	}

	if !strings.Contains(body, "/users") {
		t.Error("handleDebugRoutes() body should contain '/users'")
	}
}

func TestApplyRouterOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []RouterOption
		wantErr bool
	}{
		{
			name:    "no options",
			opts:    []RouterOption{},
			wantErr: false,
		},
		{
			name:    "single option",
			opts:    []RouterOption{WithPing()},
			wantErr: false,
		},
		{
			name:    "multiple options",
			opts:    []RouterOption{WithPing(), WithDebugRoutes()},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			err := ApplyRouterOptions(r, tt.opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyRouterOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultipleOptionsIntegration(t *testing.T) {
	r := chi.NewRouter()

	// Apply both options
	if err := ApplyRouterOptions(r, WithPing(), WithDebugRoutes()); err != nil {
		t.Fatalf("ApplyRouterOptions() error = %v", err)
	}

	// Test ping
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ping status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Test debug routes
	req = httptest.NewRequest(http.MethodGet, "/debug/routes", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("debug routes status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Debug routes should list ping endpoint
	if !strings.Contains(rec.Body.String(), "/ping") {
		t.Error("debug routes should list /ping endpoint")
	}
}

func TestNewRouter(t *testing.T) {
	logger := log.NewTestLogger("error")
	r := NewRouter(logger, WithPing(), WithDebugRoutes())

	// Test that ping endpoint works
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("NewRouter ping status = %d, want %d", rec.Code, http.StatusOK)
	}

	want := `{"status":"ok"}`
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("NewRouter ping body = %q, want to contain %q", rec.Body.String(), want)
	}

	// Test that debug routes endpoint works
	req = httptest.NewRequest(http.MethodGet, "/debug/routes", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("NewRouter debug routes status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), "/ping") {
		t.Error("NewRouter debug routes should list /ping endpoint")
	}
}
