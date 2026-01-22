package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
	if rl.limit != 10 {
		t.Errorf("expected limit 10, got %d", rl.limit)
	}
	if rl.window != time.Minute {
		t.Errorf("expected window 1m, got %v", rl.window)
	}
}

func TestRateLimiterAllow(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		window    time.Duration
		requests  int
		wantAllow bool
	}{
		{
			name:      "under limit",
			limit:     5,
			window:    time.Minute,
			requests:  3,
			wantAllow: true,
		},
		{
			name:      "at limit boundary",
			limit:     5,
			window:    time.Minute,
			requests:  5,
			wantAllow: true,
		},
		{
			name:      "over limit",
			limit:     5,
			window:    time.Minute,
			requests:  6,
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RateLimiter{
				requests: make(map[string][]time.Time),
				limit:    tt.limit,
				window:   tt.window,
			}

			var lastResult bool
			for i := 0; i < tt.requests; i++ {
				lastResult = rl.Allow("192.168.1.1")
			}

			if lastResult != tt.wantAllow {
				t.Errorf("Allow() = %v, want %v after %d requests", lastResult, tt.wantAllow, tt.requests)
			}
		})
	}
}

func TestRateLimiterAllowDifferentIPs(t *testing.T) {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    2,
		window:   time.Minute,
	}

	if !rl.Allow("192.168.1.1") {
		t.Error("first request from IP1 should be allowed")
	}
	if !rl.Allow("192.168.1.1") {
		t.Error("second request from IP1 should be allowed")
	}
	if rl.Allow("192.168.1.1") {
		t.Error("third request from IP1 should be denied")
	}

	if !rl.Allow("192.168.1.2") {
		t.Error("first request from IP2 should be allowed")
	}
	if !rl.Allow("192.168.1.2") {
		t.Error("second request from IP2 should be allowed")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    2,
		window:   50 * time.Millisecond,
	}

	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")

	if rl.Allow("192.168.1.1") {
		t.Error("should be rate limited")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("192.168.1.1") {
		t.Error("should be allowed after window expiry")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    2,
		window:   time.Minute,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimit(rl)
	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		remoteAddr     string
		wantStatusCode int
	}{
		{
			name:           "first request allowed",
			remoteAddr:     "192.168.1.1:12345",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "second request allowed",
			remoteAddr:     "192.168.1.1:12345",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "third request rate limited",
			remoteAddr:     "192.168.1.1:12345",
			wantStatusCode: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		wantIP     string
	}{
		{
			name:       "X-Forwarded-For single IP",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195"},
			wantIP:     "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			wantIP:     "203.0.113.195",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Real-IP": "203.0.113.195"},
			wantIP:     "203.0.113.195",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{},
			wantIP:     "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			headers:    map[string]string{},
			wantIP:     "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "127.0.0.1:8080",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195",
				"X-Real-IP":       "10.0.0.1",
			},
			wantIP: "203.0.113.195",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := getClientIP(req)
			if got != tt.wantIP {
				t.Errorf("getClientIP() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}
