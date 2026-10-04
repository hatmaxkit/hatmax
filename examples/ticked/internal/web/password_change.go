// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/middleware"
)

type passwordChangeService interface {
	ChangePassword(context.Context, string, string, auth.PasswordChangePolicy) error
}

// PasswordChangeHandler binds immutable server policy and uses the same finite
// ingress limits as mailbox forms. The request never supplies an account ID.
type PasswordChangeHandler struct {
	service   passwordChangeService
	policy    auth.PasswordChangePolicy
	admission MailboxHandler
}

func NewPasswordChangeHandler(service passwordChangeService, policy auth.PasswordChangePolicy) (*PasswordChangeHandler, error) {
	if policy.Requirement.Proof == 0 {
		policy.Requirement.Proof = auth.RequirePhishingResistantMFA
	}

	if policy.Requirement.MaxAge == 0 {
		policy.Requirement.MaxAge = 5 * time.Minute
	}

	if service == nil || policy.Check() != nil {
		return nil, errors.New("password change service and trusted policy are required")
	}

	return &PasswordChangeHandler{service: service, policy: policy, admission: MailboxHandler{windows: make(map[string]mailboxWindow)}}, nil
}
func (h *PasswordChangeHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Referrer-Policy", "no-referrer")
				next.ServeHTTP(w, r)
			})
		})
		r.Use(middleware.RequireSameOrigin)
		r.Get("/account/password", func(w http.ResponseWriter, r *http.Request) { mailboxPage(w, passwordChangePage) })
		r.Post("/account/password", h.change)
	})
}
func (h *PasswordChangeHandler) change(w http.ResponseWriter, r *http.Request) {
	if !h.admission.admit(r) {
		http.Error(w, "Request unavailable", http.StatusTooManyRequests)

		return
	}
	defer h.admission.release()

	password, err := passwordChangeField(w, r)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)

		return
	}

	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		err = h.service.ChangePassword(r.Context(), cookie.Value, password, h.policy)
	}

	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, auth.ErrRecoveryAttempts) || errors.Is(err, auth.ErrRecoveryCapacity) {
			status = http.StatusTooManyRequests
		}

		http.Error(w, "Password change unavailable", status)

		return
	}

	auth.ClearSessionCookie(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Password changed. Sign in again.\n"))
}

const passwordChangePage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Change password</title><h1>Change password</h1><p>Authenticate recently before changing your password. All sessions will be signed out.</p><form method="post" action="/account/password"><label>New password <input type="password" name="password" maxlength="1024" autocomplete="new-password" required></label><button>Change password</button></form></html>`

// Password candidates retain the complete Unicode input, including whitespace.
func passwordChangeField(w http.ResponseWriter, r *http.Request) (string, error) {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.URL.RawQuery != "" {
		return "", auth.ErrRecoveryUnavailable
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	err := r.ParseForm()
	if err != nil || len(r.PostForm) != 1 || len(r.PostForm["password"]) != 1 {
		return "", auth.ErrRecoveryUnavailable
	}

	password := r.PostForm.Get("password")
	if len(password) > 4096 {
		return "", auth.ErrPasswordTooLong
	}

	return password, nil
}
