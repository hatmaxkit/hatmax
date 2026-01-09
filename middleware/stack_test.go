package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultStack(t *testing.T) {
	stack := DefaultStack()

	if len(stack) != 4 {
		t.Errorf("DefaultStack() returned %d middlewares, want 4", len(stack))
	}
}

func TestDefaultInternal(t *testing.T) {
	stack := DefaultInternal()

	if len(stack) != 5 {
		t.Errorf("DefaultInternal() returned %d middlewares, want 5 (DefaultStack + InternalOnly)", len(stack))
	}
}

func TestInternalOnly(t *testing.T) {
	handler := InternalOnly()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		wantStatus int
	}{
		{"localhost", "127.0.0.1:1234", http.StatusOK},
		{"localhost IPv6", "[::1]:1234", http.StatusOK},
		{"private 10.x", "10.0.0.1:1234", http.StatusOK},
		{"private 172.16.x", "172.16.0.1:1234", http.StatusOK},
		{"private 192.168.x", "192.168.1.1:1234", http.StatusOK},
		{"public IP", "8.8.8.8:1234", http.StatusForbidden},
		{"public IP 2", "1.1.1.1:1234", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("InternalOnly() status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestIsInternalIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		{"localhost", "127.0.0.1", true},
		{"localhost with port", "127.0.0.1:8080", true},
		{"localhost IPv6", "::1", true},
		{"localhost IPv6 with port", "[::1]:8080", true},
		{"private 10.x", "10.0.0.1", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 172.31.x", "172.31.255.255", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"public IP 2", "1.1.1.1", false},
		{"public IP 3", "172.32.0.1", false},
		{"IPv6 ULA", "fc00::1", true},
		{"IPv6 ULA 2", "fd00::1", true},
		{"invalid IP", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInternalIP(tt.remoteAddr)
			if got != tt.want {
				t.Errorf("isInternalIP(%q) = %v, want %v", tt.remoteAddr, got, tt.want)
			}
		})
	}
}
