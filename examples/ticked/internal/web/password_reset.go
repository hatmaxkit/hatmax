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

type passwordResetService interface {
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string) error
}

// PasswordResetHandler owns neutral bounded initiation and explicit completion.
// Operator policy and roles belong to this application, never the core engine.
type PasswordResetHandler struct {
	service   passwordResetService
	validator middleware.SessionValidator
	operator  auth.AccessRequirement
	admission MailboxHandler
}

func NewPasswordResetHandler(service passwordResetService, validator middleware.SessionValidator, operator auth.AccessRequirement) (*PasswordResetHandler, error) {
	if service == nil || validator == nil || operator.Check(30*24*time.Hour) != nil || operator.Proof < auth.RequireMFA || operator.MaxAge <= 0 || operator.MaxAge > 5*time.Minute {
		return nil, errors.New("reset service and trusted recent operator policy are required")
	}

	return &PasswordResetHandler{service: service, validator: validator, operator: operator, admission: MailboxHandler{windows: make(map[string]mailboxWindow), acknowledgment: 6 * time.Second}}, nil
}
func (h *PasswordResetHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Referrer-Policy", "no-referrer")
				next.ServeHTTP(w, r)
			})
		})
		r.Use(middleware.RequireSameOrigin)
		r.Get("/account/password/reset", func(w http.ResponseWriter, r *http.Request) { mailboxPage(w, passwordResetRequestPage) })
		r.Get("/account/password/reset/confirm", func(w http.ResponseWriter, r *http.Request) { mailboxPage(w, passwordResetConfirmPage) })
		r.Post("/account/password/reset", h.request)
		r.Post("/account/password/reset/confirm", h.complete)
		r.Group(func(admin chi.Router) {
			admin.Use(h.operatorAdmission)
			admin.Use(middleware.RequireRoles(h.validator, h.operator, auth.NoActivity, "admin", "superadmin"))
			admin.Get("/admin/password-reset", func(w http.ResponseWriter, r *http.Request) { mailboxPage(w, operatorResetRequestPage) })
			admin.Post("/admin/password-reset", h.requestAdmitted)
		})
	})
}
func (h *PasswordResetHandler) request(w http.ResponseWriter, r *http.Request) {
	if !h.admission.admit(r) {
		http.Error(w, "Request unavailable", http.StatusTooManyRequests)

		return
	}
	defer h.admission.release()

	h.requestAdmitted(w, r)
}
func (h *PasswordResetHandler) requestAdmitted(w http.ResponseWriter, r *http.Request) {
	mailbox, err := mailboxField(w, r, "email", 254)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)

		return
	}

	start := time.Now()
	work, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	_ = h.service.RequestPasswordReset(work, mailbox)

	cancel()

	timer := time.NewTimer(time.Until(start.Add(h.admission.acknowledgment)))
	defer timer.Stop()

	select {
	case <-r.Context().Done():
		return
	case <-timer.C:
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("If this account is eligible, a password reset message will be sent.\n"))
}
func (h *PasswordResetHandler) complete(w http.ResponseWriter, r *http.Request) {
	if !h.admission.admit(r) {
		http.Error(w, "Request unavailable", http.StatusTooManyRequests)

		return
	}
	defer h.admission.release()

	token, password, err := passwordResetFields(w, r)
	if err == nil {
		err = h.service.ResetPassword(r.Context(), token, password)
	}

	if err != nil {
		http.Error(w, "Password reset unavailable", http.StatusForbidden)

		return
	}

	auth.ClearSessionCookie(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Password reset. Sign in again with any required MFA.\n"))
}
func passwordResetFields(w http.ResponseWriter, r *http.Request) (string, string, error) {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.URL.RawQuery != "" {
		return "", "", auth.ErrRecoveryUnavailable
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	err := r.ParseForm()
	if err != nil || len(r.PostForm) != 2 || len(r.PostForm["token"]) != 1 || len(r.PostForm["password"]) != 1 {
		return "", "", auth.ErrRecoveryUnavailable
	}

	token, password := r.PostForm.Get("token"), r.PostForm.Get("password")
	if len(token) != 80 || len(password) > 4096 {
		return "", "", auth.ErrRecoveryUnavailable
	}

	return token, password, nil
}

const passwordResetRequestPage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Reset password</title><h1>Reset password</h1><form method="post" action="/account/password/reset"><label>Current account email <input type="email" name="email" maxlength="254" required></label><button>Send reset link</button></form></html>`
const operatorResetRequestPage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Request account reset</title><h1>Request account reset</h1><p>The account must already have a verified current mailbox. The account holder chooses the password and still needs required MFA.</p><form method="post" action="/admin/password-reset"><label>Current account email <input type="email" name="email" maxlength="254" required></label><button>Send reset link</button></form></html>`
const passwordResetConfirmPage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Reset password</title><h1>Reset password</h1><form method="post" action="/account/password/reset/confirm"><label>Reset token <input id="token" name="token" maxlength="80" autocomplete="off" required></label><label>New password <input type="password" name="password" autocomplete="new-password" required></label><button>Reset password</button></form><script>const value=new URLSearchParams(location.hash.slice(1)).get('token');history.replaceState(null,'',location.pathname);if(value)document.getElementById('token').value=value;</script></html>`

// Operator POST admission runs before session/account validation too.
func (h *PasswordResetHandler) operatorAdmission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !h.admission.admit(r) {
				http.Error(w, "Request unavailable", http.StatusTooManyRequests)

				return
			}
			defer h.admission.release()
		}

		next.ServeHTTP(w, r)
	})
}
