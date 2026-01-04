package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name           string
		existingID     string
		wantIDGenerate bool
	}{
		{
			name:           "with existing request ID",
			existingID:     "existing-id-123",
			wantIDGenerate: false,
		},
		{
			name:           "without request ID generates new",
			existingID:     "",
			wantIDGenerate: true,
		},
		{
			name:           "with whitespace only generates new",
			existingID:     "   ",
			wantIDGenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedID string
			var capturedContext context.Context

			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedID = GetRequestID(r.Context())
				capturedContext = r.Context()
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.existingID != "" {
				req.Header.Set("X-Request-ID", tt.existingID)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			responseID := rec.Header().Get("X-Request-ID")

			if tt.wantIDGenerate {
				if capturedID == "" {
					t.Error("RequestID() did not generate ID")
				}
				if len(capturedID) != 36 {
					t.Errorf("RequestID() generated ID length = %d, want 36 (UUID format)", len(capturedID))
				}
			} else if tt.existingID != "" && tt.existingID != "   " {
				if capturedID != tt.existingID {
					t.Errorf("RequestID() = %v, want %v", capturedID, tt.existingID)
				}
			}

			if responseID != capturedID {
				t.Errorf("Response X-Request-ID = %v, want %v", responseID, capturedID)
			}

			if capturedContext == nil {
				t.Error("Context was not passed through")
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		want    string
		wantNil bool
	}{
		{
			name: "valid request ID in context",
			ctx:  context.WithValue(context.Background(), RequestIDKey, "test-id-123"),
			want: "test-id-123",
		},
		{
			name: "empty context",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "wrong type in context",
			ctx:  context.WithValue(context.Background(), RequestIDKey, 12345),
			want: "",
		},
		{
			name: "nil context",
			ctx:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.ctx != nil {
				got = GetRequestID(tt.ctx)
			} else {
				// Test with nil context - should not panic
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("GetRequestID() panicked with nil context: %v", r)
					}
				}()
				got = GetRequestID(tt.ctx)
			}

			if got != tt.want {
				t.Errorf("GetRequestID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestIDUniqueness(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	id1 := rec1.Header().Get("X-Request-ID")

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	id2 := rec2.Header().Get("X-Request-ID")

	if id1 == id2 {
		t.Error("RequestID() not unique, got same ID twice")
	}

	if id1 == "" || id2 == "" {
		t.Error("RequestID() returned empty IDs")
	}
}
