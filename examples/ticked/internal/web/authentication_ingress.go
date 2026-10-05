// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/middleware"
)

type ingressKey struct{}
type ingressRequest struct{ workDeadline, ackDeadline time.Time }

// AuthenticationIngress owns one shared finite HTTP boundary and no workers.
type AuthenticationIngress struct {
	mu       sync.Mutex
	peers    *middleware.RateLimiter
	settings config.AuthenticationIngressSettings
	active   int
	closed   bool
}

// NewAuthenticationIngress validates a shared boundary without starting workers.
func NewAuthenticationIngress(cfg *config.Config) (*AuthenticationIngress, error) {
	s, err := cfg.AuthenticationIngressSettings()
	if err != nil {
		return nil, err
	}

	peers, err := middleware.NewRateLimiter(middleware.RateLimitConfig{Limit: s.PeerRequests, Window: s.PeerWindow, MaxPeers: s.MaxPeers, CleanupBatch: s.CleanupBatch})
	if err != nil {
		return nil, err
	}

	return &AuthenticationIngress{peers: peers, settings: s}, nil
}

// Close prevents new admissions before owned infrastructure is released.
func (g *AuthenticationIngress) Close() { g.mu.Lock(); g.closed = true; g.mu.Unlock() }

// Cleanup inspects one bounded peer batch without evicting live work.
func (g *AuthenticationIngress) Cleanup() int { return g.peers.Cleanup() }
func authenticationMutation(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	return authenticationPath(r.URL.Path)
}
func authenticationPath(path string) bool {
	switch path {
	case "/signup", "/signin", "/signout", "/reauthenticate", "/sessions/revoke", "/admin/password-reset":
		return true
	}

	return strings.HasPrefix(path, "/authenticators/") || path == "/account/mailbox" || strings.HasPrefix(path, "/account/mailbox/") || path == "/account/password" || strings.HasPrefix(path, "/account/password/")
}

// AuthenticationLog omits URI logging for authentication routes, including GET.
// Terminal security observations have their own typed, redacted contract.
func AuthenticationLog(next http.Handler) http.Handler {
	return authenticationLog(chimiddleware.Logger, next)
}
func authenticationLog(requestLog func(http.Handler) http.Handler, next http.Handler) http.Handler {
	logged := requestLog(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authenticationPath(r.URL.Path) {
			next.ServeHTTP(w, r)

			return
		}

		logged.ServeHTTP(w, r)
	})
}

// Middleware runs after ProxyHeaders, before body parsing and account work.
func (g *AuthenticationIngress) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authenticationMutation(r) {
			next.ServeHTTP(w, r)

			return
		}

		accepted := time.Now()

		g.mu.Lock()

		status := 0

		if g.closed || r.Context().Err() != nil {
			status = http.StatusServiceUnavailable
		} else if g.active >= g.settings.MaxActive || !g.peers.Allow(middleware.ClientIP(r)) {
			status = http.StatusTooManyRequests
		} else {
			g.active++
		}
		g.mu.Unlock()

		if status != 0 {
			http.Error(w, "Request unavailable", status)

			return
		}

		defer func() { g.mu.Lock(); g.active--; g.mu.Unlock() }()

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")

		entry := ingressRequest{workDeadline: accepted.Add(g.settings.WorkTimeout), ackDeadline: accepted.Add(g.settings.Acknowledgment)}
		controller := http.NewResponseController(w)
		// The server resets this read deadline between requests, after body drain.
		_ = controller.SetReadDeadline(entry.workDeadline)

		middleware.RequireSameOrigin(next).ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ingressKey{}, entry)))
	})
}
func publicAuthenticationWork(w http.ResponseWriter, r *http.Request) (context.Context, context.CancelFunc, bool) {
	entry, ok := r.Context().Value(ingressKey{}).(ingressRequest)
	if !ok {
		http.Error(w, "Request unavailable", http.StatusServiceUnavailable)

		return nil, func() {}, false
	}

	ctx, cancel := context.WithDeadline(r.Context(), entry.workDeadline)

	return ctx, cancel, true
}
func waitAuthentication(r *http.Request) bool {
	entry, ok := r.Context().Value(ingressKey{}).(ingressRequest)
	if !ok {
		return false
	}

	timer := time.NewTimer(time.Until(entry.ackDeadline))
	defer timer.Stop()

	select {
	case <-r.Context().Done():
		return false
	case <-timer.C:
		return r.Context().Err() == nil
	}
}
func publicPasswordDenial(w http.ResponseWriter, r *http.Request) {
	if !waitAuthentication(r) {
		http.Error(w, "Request unavailable", http.StatusServiceUnavailable)

		return
	}

	http.Error(w, "Authentication unavailable", http.StatusForbidden)
}

// credentialForm rejects ambiguous and oversized public input before account work.
func credentialForm(w http.ResponseWriter, r *http.Request, signup bool) (string, string, error) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/x-www-form-urlencoded" || r.URL.RawQuery != "" {
		return "", "", errors.New("invalid credential form")
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	err = r.ParseForm()
	if err != nil {
		return "", "", errors.New("invalid credential form")
	}

	fields := 2
	if signup {
		fields = 3
	}

	if len(r.PostForm) != fields {
		return "", "", errors.New("invalid credential form")
	}

	for key, values := range r.PostForm {
		if len(values) != 1 || (key != "email" && key != "password" && (!signup || key != "confirm_password")) {
			return "", "", errors.New("invalid credential form")
		}
	}

	email, password := r.PostForm.Get("email"), r.PostForm.Get("password")
	if auth.CheckCredentialIdentity(email) != nil || password == "" || len(password) > 4096 || !utf8.ValidString(password) || signup && password != r.PostForm.Get("confirm_password") {
		return "", "", errors.New("invalid credential form")
	}

	return email, password, nil
}
