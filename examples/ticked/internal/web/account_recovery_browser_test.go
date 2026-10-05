//go:build browser

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	featureauth "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/mailer"
)

// Extend the actual authenticator journeys through production recovery forms;
// captured mail and a dropped committed response never manufacture authority.
func TestAccountRecoveryBrowser(t *testing.T) {
	runAuthenticatorBrowser(t, true)
}

type recoveryBrowserFixture struct {
	mu      sync.Mutex
	links   map[string]string
	failure bool
	notices int
	lost    atomic.Int32
}

func newRecoveryBrowserFixture() *recoveryBrowserFixture {
	return &recoveryBrowserFixture{links: make(map[string]string)}
}

func (f *recoveryBrowserFixture) Send(_ context.Context, m *mailer.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	linked := false

	for _, line := range strings.Split(m.Text, "\n") {
		for _, purpose := range []string{"mailbox", "password/reset"} {
			if strings.Contains(line, "/account/"+purpose+"/confirm#token=") {
				f.links[purpose] = line
				linked = true
			}
		}
	}

	if !linked {
		f.notices++
	}

	if f.failure {
		return errors.New("captured provider failure")
	}

	return nil
}

func (f *recoveryBrowserFixture) register(t *testing.T, r chi.Router, base *auth.Service, q *featureauth.Queries, origin string, strong auth.AccessRequirement) {
	t.Helper()

	logger := log.NewTestLogger("error")

	service, err := auth.NewRecoveryService(base, q, config.RecoveryConfig{}, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}

	delivery, err := featureauth.NewMailboxDelivery(service, q, f, logger, origin, true)
	if err != nil {
		t.Fatal(err)
	}

	mailbox, err := NewMailboxHandler(delivery)
	if err != nil {
		t.Fatal(err)
	}

	change, err := NewPasswordChangeHandler(delivery, auth.PasswordChangePolicy{Requirement: strong, AllowPassword: true})
	if err != nil {
		t.Fatal(err)
	}

	reset, err := NewPasswordResetHandler(delivery, base, strong)
	if err != nil {
		t.Fatal(err)
	}

	user, err := q.GetUserByEmail(t.Context(), "passkey@example.com")
	if err != nil {
		t.Fatal(err)
	}
	// Establish the fixture's operator role through the production update before
	// any browser authentication; requests still prove current role and MFA.
	err = q.UpdateUserRoles(t.Context(), user.ID, []string{"admin"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	mailbox.RegisterRoutes(r)
	change.RegisterRoutes(r)
	reset.RegisterRoutes(r)
	// These test-only endpoints expose captured mail and provider controls solely
	// to the owned fixture. They never approve proof, mint tokens or mutate factors.
	r.Get("/__test/recovery-mail", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			Link    string `json:"link"`
			Notices int    `json:"notices"`
			Lost    int32  `json:"lost"`
		}{f.links[r.URL.Query().Get("purpose")], f.notices, f.lost.Load()})
	})
	r.Post("/__test/mail-mode", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Failure bool `json:"failure"`
		}

		r.Body = http.MaxBytesReader(w, r.Body, 128)

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, "Invalid fixture input", 400)
			return
		}

		f.mu.Lock()
		f.failure = input.Failure
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	r.Post("/__test/retry-notices", func(w http.ResponseWriter, r *http.Request) {
		err := delivery.DispatchMailboxNotices(r.Context(), user.ID, 5)
		if err != nil {
			http.Error(w, "Fixture retry failed", 500)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

// Only the owned test transport loses a response, after the production handler
// has finished its real transaction. An unsuccessful handler response is retained.
func (f *recoveryBrowserFixture) loseResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/password/reset/confirm" || r.Header.Get("X-Test-Lose-Response") != "yes" {
			next.ServeHTTP(w, r)

			return
		}

		reply := httptest.NewRecorder()
		next.ServeHTTP(reply, r)

		if reply.Code != http.StatusOK {
			for k, values := range reply.Header() {
				w.Header()[k] = values
			}

			w.WriteHeader(reply.Code)
			_, _ = w.Write(reply.Body.Bytes())

			return
		}

		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			http.Error(w, "Fixture transport failed", 500)
			return
		}

		f.lost.Add(1)

		_ = conn.Close()
	})
}
