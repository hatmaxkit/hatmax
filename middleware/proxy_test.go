// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

// TestInternalSpoof ensures default internal stacks cannot authorize header identities.
func TestInternalSpoof(t *testing.T) {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		t.Run(header, func(t *testing.T) {
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			stack := DefaultInternal()

			for i := len(stack) - 1; i >= 0; i-- {
				handler = stack[i](handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.0.2.10:1234"
			req.Header.Set(header, "127.0.0.1")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("forged %s: status %d, want 403", header, rec.Code)
			}
		})
	}
}

// TestProxyHeaders verifies that only approved hops can supply a canonical client IP.
func TestProxyHeaders(t *testing.T) {
	tests := []struct {
		name    string
		peer    string
		xff     []string
		xri     []string
		trusted bool
		want    string
	}{
		{"default ignores forwarded", "10.0.0.2:1234", []string{"203.0.113.1"}, nil, false, "10.0.0.2"},
		{"default ignores real", "10.0.0.2:1234", nil, []string{"203.0.113.1"}, false, "10.0.0.2"},
		{"untrusted peer", "192.0.2.10:1234", []string{"127.0.0.1"}, []string{"10.0.0.1"}, true, "192.0.2.10"},
		{"trusted single hop", "10.0.0.2:1234", []string{"203.0.113.1"}, nil, true, "203.0.113.1"},
		{"trusted chain", "10.0.0.2:1234", []string{"203.0.113.1, 10.0.0.3"}, nil, true, "203.0.113.1"},
		{"forged left prefix", "10.0.0.2:1234", []string{"127.0.0.1, 203.0.113.1"}, nil, true, "203.0.113.1"},
		{"untrusted intermediary", "10.0.0.2:1234", []string{"203.0.113.1, 192.0.2.4, 10.0.0.3"}, nil, true, "192.0.2.4"},
		{"all hops trusted", "10.0.0.2:1234", []string{"10.0.0.4, 10.0.0.3"}, nil, true, "10.0.0.4"},
		{"multiple header lines", "10.0.0.2:1234", []string{"127.0.0.1, 203.0.113.1", "10.0.0.3"}, nil, true, "203.0.113.1"},
		{"whitespace", "10.0.0.2:1234", []string{" 203.0.113.1 , 10.0.0.3 "}, nil, true, "203.0.113.1"},
		{"invalid trusted hop", "10.0.0.2:1234", []string{"203.0.113.1, invalid, 10.0.0.3"}, nil, true, "10.0.0.2"},
		{"invalid left prefix ignored", "10.0.0.2:1234", []string{"invalid, 203.0.113.1"}, nil, true, "203.0.113.1"},
		{"empty header", "10.0.0.2:1234", []string{""}, []string{"203.0.113.1"}, true, "10.0.0.2"},
		{"empty right hop", "10.0.0.2:1234", []string{"203.0.113.1,"}, nil, true, "10.0.0.2"},
		{"hostname", "10.0.0.2:1234", []string{"example.com"}, nil, true, "10.0.0.2"},
		{"port in header", "10.0.0.2:1234", []string{"203.0.113.1:1234"}, nil, true, "10.0.0.2"},
		{"CIDR in header", "10.0.0.2:1234", []string{"203.0.113.0/24"}, nil, true, "10.0.0.2"},
		{"zone in header", "10.0.0.2:1234", []string{"fe80::1%eth0"}, nil, true, "10.0.0.2"},
		{"invalid forwarded overrides real", "10.0.0.2:1234", []string{"invalid"}, []string{"203.0.113.1"}, true, "10.0.0.2"},
		{"forwarded precedence", "10.0.0.2:1234", []string{"203.0.113.1"}, []string{"203.0.113.2"}, true, "203.0.113.1"},
		{"trusted real IP", "10.0.0.2:1234", nil, []string{"203.0.113.1"}, true, "203.0.113.1"},
		{"invalid real IP", "10.0.0.2:1234", nil, []string{"invalid"}, true, "10.0.0.2"},
		{"multiple real IP lines", "10.0.0.2:1234", nil, []string{"203.0.113.1", "203.0.113.2"}, true, "10.0.0.2"},
		{"real IP list", "10.0.0.2:1234", nil, []string{"203.0.113.1, 203.0.113.2"}, true, "10.0.0.2"},
		{"trusted IPv6 proxy", "[fd12::2]:1234", []string{"2001:0db8:0:0::1"}, nil, true, "2001:db8::1"},
		{"IPv6 chain", "[fd12::2]:1234", []string{"2001:db8::1, fd34::3"}, nil, true, "2001:db8::1"},
		{"IPv6 brackets in header", "[fd12::2]:1234", []string{"[2001:db8::1]:1234"}, nil, true, "fd12::2"},
		{"mapped IPv4 peer", "[::ffff:10.0.0.2]:1234", []string{"::ffff:203.0.113.1"}, nil, true, "203.0.113.1"},
		{"IPv6 peer without port", "2001:0db8::1", nil, nil, false, "2001:db8::1"},
		{"IPv6 peer zone", "[fe80::1%eth0]:1234", nil, nil, false, "fe80::1"},
		{"bare IPv4 peer", "192.0.2.10", nil, nil, false, "192.0.2.10"},
		{"invalid peer", "invalid", []string{"203.0.113.1"}, nil, true, "invalid"},
		{"empty peer", "", []string{"203.0.113.1"}, nil, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var trusted []netip.Prefix

			if tt.trusted {
				trusted = []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fc00::/7")}
			}

			handler := ProxyHeaders(trusted...)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if got := ClientIP(r); got != tt.want {
					t.Errorf("ClientIP = %q, want %q", got, tt.want)
				}

				if r.RemoteAddr != tt.peer {
					t.Errorf("RemoteAddr changed: %q, want %q", r.RemoteAddr, tt.peer)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.peer

			for _, value := range tt.xff {
				req.Header.Add("X-Forwarded-For", value)
			}

			for _, value := range tt.xri {
				req.Header.Add("X-Real-IP", value)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}

// TestProxyPolicyCopy prevents caller-owned slices from changing an installed policy.
func TestProxyPolicyCopy(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	policy := ProxyHeaders(trusted...)
	trusted[0] = netip.MustParsePrefix("192.0.2.0/24")
	called := false
	check := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true

		if got := ClientIP(r); got != "203.0.113.1" {
			t.Errorf("ClientIP = %q, policy changed after construction", got)
		}
	})
	// Reinstalling the policy must not replace the original peer with a rewritten address.
	handler := policy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = "192.0.2.10:4321"
		policy(check).ServeHTTP(w, r)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("downstream handler was not called")
	}
}

// TestInternalPeer keeps network restriction separate from forwarded client identity.
func TestInternalPeer(t *testing.T) {
	tests := []struct {
		name   string
		peer   string
		client string
		status int
	}{
		{"approved public proxy still denied", "192.0.2.10:1234", "127.0.0.1", http.StatusForbidden},
		{"private proxy with public client", "10.0.0.2:1234", "203.0.113.1", http.StatusOK},
		{"IPv6 private peer", "[fd12:3456::1]:1234", "2001:db8::1", http.StatusOK},
		{"mapped IPv4 private peer", "[::ffff:10.0.0.2]:1234", "203.0.113.1", http.StatusOK},
		{"IPv6 public peer", "[2001:db8::2]:1234", "::1", http.StatusForbidden},
		{"invalid peer", "invalid", "127.0.0.1", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			stack := DefaultInternal(netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0"))

			for i := len(stack) - 1; i >= 0; i-- {
				handler = stack[i](handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.peer
			req.Header.Set("X-Forwarded-For", tt.client)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("status %d, want %d", rec.Code, tt.status)
			}
		})
	}
}

// TestPeerSnapshot prevents later middleware rewrites from bypassing InternalOnly.
func TestPeerSnapshot(t *testing.T) {
	guard := InternalOnly()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler := ProxyHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Forwarded-For", "127.0.0.1")

		if got := ClientIP(r); got != "192.0.2.10" {
			t.Errorf("ClientIP changed after snapshot: %q", got)
		}

		guard.ServeHTTP(w, r)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("rewritten peer: status %d, want 403", rec.Code)
	}
}

// TestRateLimitIdentity verifies proxy isolation and equivalent IPv4/IPv6 buckets.
func TestRateLimitIdentity(t *testing.T) {
	tests := []struct {
		name       string
		peers      [2]string
		clients    [2]string
		policy     bool
		wantSecond int
	}{
		{"approved distinct clients", [2]string{"10.0.0.2:1234", "10.0.0.2:1234"}, [2]string{"203.0.113.1", "203.0.113.2"}, true, http.StatusOK},
		{"unapproved header changes", [2]string{"192.0.2.10:1234", "192.0.2.10:4321"}, [2]string{"203.0.113.1", "203.0.113.2"}, true, http.StatusTooManyRequests},
		{"proxy without policy", [2]string{"10.0.0.2:1234", "10.0.0.2:1234"}, [2]string{"203.0.113.1", "203.0.113.2"}, false, http.StatusTooManyRequests},
		{"mapped client same bucket", [2]string{"10.0.0.2:1234", "10.0.0.2:1234"}, [2]string{"203.0.113.1", "::ffff:203.0.113.1"}, true, http.StatusTooManyRequests},
		{"IPv6 clients", [2]string{"10.0.0.2:1234", "10.0.0.2:1234"}, [2]string{"2001:db8::1", "2001:0db8:0:0::1"}, true, http.StatusTooManyRequests},
		{"IPv6 peer ports", [2]string{"[2001:db8::1]:1234", "[2001:db8::1]:4321"}, [2]string{}, false, http.StatusTooManyRequests},
		{"bare IPv6 peers", [2]string{"2001:db8::1", "2001:db8::2"}, [2]string{}, false, http.StatusOK},
		{"mapped peer same bucket", [2]string{"192.0.2.10:1234", "[::ffff:192.0.2.10]:4321"}, [2]string{}, false, http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := rateLimiterForTest(t, RateLimitConfig{Limit: 1, Window: time.Minute})
			handler := RateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			if tt.policy {
				handler = ProxyHeaders(netip.MustParsePrefix("10.0.0.0/8"))(handler)
			}

			for i, peer := range tt.peers {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = peer
				req.Header.Set("X-Forwarded-For", tt.clients[i])

				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				want := http.StatusOK

				if i == 1 {
					want = tt.wantSecond
				}

				if rec.Code != want {
					t.Fatalf("request %d: status %d, want %d", i+1, rec.Code, want)
				}
			}
		})
	}
}

// TestRateLimitSpoof ensures changing untrusted headers cannot reset the peer's bucket.
func TestRateLimitSpoof(t *testing.T) {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		t.Run(header, func(t *testing.T) {
			limiter := rateLimiterForTest(t, RateLimitConfig{Limit: 1, Window: time.Minute})
			handler := RateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for i, value := range []string{"203.0.113.1", "203.0.113.2"} {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "192.0.2.10:1234"
				req.Header.Set(header, value)

				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				want := http.StatusOK

				if i == 1 {
					want = http.StatusTooManyRequests
				}

				if rec.Code != want {
					t.Fatalf("request %d: status %d, want %d", i+1, rec.Code, want)
				}
			}
		})
	}
}

// FuzzProxyHeaders checks that arbitrary headers cannot alter the peer or create invalid identities.
func FuzzProxyHeaders(f *testing.F) {
	for _, value := range []string{"", "invalid", "203.0.113.1", "127.0.0.1, 203.0.113.1", "2001:db8::1, 10.0.0.3", "fe80::1%eth0"} {
		f.Add(value, value)
	}

	f.Fuzz(func(t *testing.T, xff, xri string) {
		for _, trusted := range [][]netip.Prefix{nil, {netip.MustParsePrefix("10.0.0.0/8")}} {
			handler := ProxyHeaders(trusted...)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if r.RemoteAddr != "10.0.0.2:1234" {
					t.Fatal("proxy headers changed RemoteAddr")
				}

				client := ClientIP(r)
				ip, err := netip.ParseAddr(client)

				if err != nil || ip.Zone() != "" || ip.Unmap().String() != client {
					t.Fatalf("noncanonical client IP: %q", client)
				}

				if len(trusted) == 0 && client != "10.0.0.2" {
					t.Fatalf("unapproved headers changed client IP: %q", client)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "10.0.0.2:1234"
			req.Header.Set("X-Forwarded-For", xff)
			req.Header.Set("X-Real-IP", xri)
			handler.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
