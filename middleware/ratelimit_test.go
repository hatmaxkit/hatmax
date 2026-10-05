// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func rateLimiterForTest(t *testing.T, cfg RateLimitConfig) *RateLimiter {
	t.Helper()

	limiter, err := NewRateLimiter(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return limiter
}

// Configuration rejects unbounded inputs rather than starting a cleanup worker.
func TestRateLimitConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  RateLimitConfig
	}{
		{"negative limit", RateLimitConfig{Limit: -1}},
		{"large limit", RateLimitConfig{Limit: 1001}},
		{"short window", RateLimitConfig{Window: time.Millisecond}},
		{"large window", RateLimitConfig{Window: 2 * time.Hour}},
		{"negative peers", RateLimitConfig{MaxPeers: -1}},
		{"large peers", RateLimitConfig{MaxPeers: 10001}},
		{"negative cleanup", RateLimitConfig{CleanupBatch: -1}},
		{"large cleanup", RateLimitConfig{CleanupBatch: 1001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewRateLimiter(tc.cfg)
			if err == nil {
				t.Fatal("invalid finite limit accepted")
			}
		})
	}
}

// Fixed windows reject overflow without evicting live peers or extending denial.
func TestRateLimitBounds(t *testing.T) {
	limiter := rateLimiterForTest(t, RateLimitConfig{Limit: 2, MaxPeers: 2, CleanupBatch: 1})
	now := time.Unix(100, 0)

	limiter.now = func() time.Time { return now }
	for _, peer := range []string{"192.0.2.1", "2001:db8::1"} {
		if !limiter.Allow(peer) || !limiter.Allow(peer) || limiter.Allow(peer) {
			t.Fatal("fixed allowance changed")
		}
	}

	if limiter.Allow("192.0.2.2") || limiter.Allow("::ffff:192.0.2.1") || limiter.Allow("2001:0db8:0:0::1") {
		t.Fatal("live capacity or canonical bucket bypassed")
	}

	if len(limiter.peers) != 2 {
		t.Fatal("live peer evicted")
	}

	now = now.Add(time.Minute)

	if !limiter.Allow("2001:db8::1") {
		t.Fatal("expired target outside cleanup cursor was not renewed")
	}

	if len(limiter.peers) > 2 {
		t.Fatal("peer capacity exceeded")
	}

	now = now.Add(-2 * time.Minute)

	if limiter.Allow("2001:db8::1") {
		t.Fatal("backward clock reset allowance")
	}
}

// Explicit retirement touches at most one batch and leaves live entries intact.
func TestRateLimitCleanup(t *testing.T) {
	limiter := rateLimiterForTest(t, RateLimitConfig{MaxPeers: 4, CleanupBatch: 1})
	now := time.Unix(100, 0)

	limiter.now = func() time.Time { return now }
	for _, peer := range []string{"192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4"} {
		limiter.Allow(peer)
	}

	if limiter.Cleanup() != 0 || len(limiter.peers) != 4 {
		t.Fatal("live peer retired")
	}

	now = now.Add(time.Minute)

	if limiter.Cleanup() != 1 || len(limiter.peers) != 3 {
		t.Fatal("cleanup exceeded one entry")
	}
}

// Concurrent callers compete for exactly one finite allowance without queues.
func TestRateLimitRace(t *testing.T) {
	limiter := rateLimiterForTest(t, RateLimitConfig{Limit: 5})

	var (
		allowed atomic.Int32
		group   sync.WaitGroup
	)
	for range 50 {
		group.Go(func() {
			if limiter.Allow("192.0.2.1") {
				allowed.Add(1)
			}
		})
	}

	group.Wait()

	if allowed.Load() != 5 {
		t.Fatal("concurrent allowance exceeded")
	}
}

// Arbitrary input cannot create invalid or unbounded peer identities.
func FuzzRateLimitPeer(f *testing.F) {
	for _, peer := range []string{"192.0.2.1", "::ffff:192.0.2.1", "2001:db8::1", "invalid", "fe80::1%eth0", ""} {
		f.Add(peer)
	}

	f.Fuzz(func(t *testing.T, peer string) {
		limiter := rateLimiterForTest(t, RateLimitConfig{Limit: 1, MaxPeers: 1})
		ip, err := netip.ParseAddr(peer)

		valid := err == nil && ip.Zone() == ""
		if limiter.Allow(peer) != valid {
			t.Fatal("peer parser disagreed with canonical address")
		}

		if valid && (limiter.Allow(ip.Unmap().String()) || limiter.Allow("192.0.2.254") && ip.Unmap().String() != "192.0.2.254") {
			t.Fatal("bucket or capacity bypassed")
		}

		if len(limiter.peers) > 1 {
			t.Fatal("peer capacity exceeded")
		}
	})
}

// TestClientIPFallback verifies that without proxy policy only the socket peer is used.
func TestClientIPFallback(t *testing.T) {
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
			wantIP:     "127.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			wantIP:     "127.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Real-IP": "203.0.113.195"},
			wantIP:     "127.0.0.1",
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
			name:       "both headers ignored without policy",
			remoteAddr: "127.0.0.1:8080",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195",
				"X-Real-IP":       "10.0.0.1",
			},
			wantIP: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := ClientIP(req)
			if got != tt.wantIP {
				t.Errorf("ClientIP() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}
