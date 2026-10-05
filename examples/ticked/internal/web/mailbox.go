// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/middleware"
)

type mailboxService interface {
	RequestMailboxVerification(context.Context, string) error
	ConfirmMailboxVerification(context.Context, string) error
}

type mailboxWindow struct {
	start    time.Time
	attempts int
}

// MailboxHandler owns finite process-local admission before any account lookup.
// Trusted proxies need their own deployment limiter; forwarded addresses are ignored.
type MailboxHandler struct {
	service        mailboxService
	mu             sync.Mutex
	windows        map[string]mailboxWindow
	active         int
	acknowledgment time.Duration
}

func NewMailboxHandler(service mailboxService) (*MailboxHandler, error) {
	if service == nil {
		return nil, errors.New("mailbox service is required")
	}

	return &MailboxHandler{service: service, windows: make(map[string]mailboxWindow), acknowledgment: 6 * time.Second}, nil
}

func (h *MailboxHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Referrer-Policy", "no-referrer")
				next.ServeHTTP(w, r)
			})
		})
		r.Use(middleware.RequireSameOrigin)
		r.Get("/account/mailbox", func(w http.ResponseWriter, r *http.Request) { mailboxPage(w, mailboxRequestPage) })
		r.Get("/account/mailbox/confirm", func(w http.ResponseWriter, r *http.Request) { mailboxPage(w, mailboxConfirmPage) })
		r.Post("/account/mailbox", h.request)
		r.Post("/account/mailbox/confirm", h.confirm)
	})
}

func mailboxPage(w http.ResponseWriter, page string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page + recoveryFormScript))
}

func (h *MailboxHandler) admit(r *http.Request) bool {
	address, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		address = r.RemoteAddr
	}

	ip := net.ParseIP(address)
	if ip == nil {
		return false
	}

	address = ip.String()

	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for key, entry := range h.windows {
		if !now.Before(entry.start.Add(time.Minute)) {
			delete(h.windows, key)
		}
	}

	entry, exists := h.windows[address]
	if h.active >= 32 || entry.attempts >= 12 || !exists && len(h.windows) >= 1024 {
		return false
	}

	if !exists {
		entry.start = now
	}

	entry.attempts++
	h.windows[address] = entry
	h.active++

	return true
}
func (h *MailboxHandler) release() { h.mu.Lock(); h.active--; h.mu.Unlock() }

func mailboxField(w http.ResponseWriter, r *http.Request, key string, max int) (string, error) {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.URL.RawQuery != "" {
		return "", auth.ErrRecoveryUnavailable
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	err := r.ParseForm()
	if err != nil || len(r.PostForm) != 1 || len(r.PostForm[key]) != 1 {
		return "", auth.ErrRecoveryUnavailable
	}

	value := r.PostForm.Get(key)
	if len(value) == 0 || len(value) > max || strings.ContainsAny(value, "\x00\r\n") {
		return "", auth.ErrRecoveryUnavailable
	}

	return value, nil
}

func (h *MailboxHandler) request(w http.ResponseWriter, r *http.Request) {
	if !h.admit(r) {
		http.Error(w, "Request unavailable", http.StatusTooManyRequests)

		return
	}
	defer h.release()

	mailbox, err := mailboxField(w, r, "email", 254)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)

		return
	}
	// Account eligibility, storage and provider outcomes share a fixed public deadline.
	start := time.Now()
	work, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	_ = h.service.RequestMailboxVerification(work, mailbox)

	cancel()

	timer := time.NewTimer(time.Until(start.Add(h.acknowledgment)))
	defer timer.Stop()

	select {
	case <-r.Context().Done():
		return
	case <-timer.C:
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("If this account is eligible, a verification message will be sent.\n"))
}

func (h *MailboxHandler) confirm(w http.ResponseWriter, r *http.Request) {
	if !h.admit(r) {
		http.Error(w, "Request unavailable", http.StatusTooManyRequests)

		return
	}
	defer h.release()

	token, err := mailboxField(w, r, "token", 80)
	if err == nil {
		err = h.service.ConfirmMailboxVerification(r.Context(), token)
	}

	if err != nil {
		http.Error(w, "Verification unavailable", http.StatusForbidden)

		return
	}

	auth.ClearSessionCookie(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Mailbox verified. Sign in again.\n"))
}

const mailboxRequestPage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Verify mailbox</title><h1>Verify mailbox</h1><form method="post" action="/account/mailbox"><label>Email <input type="email" name="email" maxlength="254" required></label><button>Send verification</button></form></html>`

// A fragment keeps the bearer out of request URLs and access logs. GET only
// presents an explicit form; navigation removes the fragment before submission.
const mailboxConfirmPage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Verify mailbox</title><h1>Verify mailbox</h1><form method="post" action="/account/mailbox/confirm"><label>Verification token <input id="token" name="token" maxlength="80" autocomplete="off" required></label><button>Verify mailbox</button></form><script>const value=new URLSearchParams(location.hash.slice(1)).get('token');history.replaceState(null,'',location.pathname);if(value)document.getElementById('token').value=value;</script></html>`

// Native no-referrer form navigation can carry an opaque Origin. A same-origin
// fetch preserves browser source validation and the exact supported wire format.
const recoveryFormScript = `<script>document.querySelector('form').addEventListener('submit',async event=>{
 event.preventDefault();const form=event.currentTarget,button=form.querySelector('button');button.disabled=true;
 try{const response=await fetch(form.action,{method:'POST',mode:'same-origin',credentials:'same-origin',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:new URLSearchParams(new FormData(form)),signal:AbortSignal.timeout(10000)});
 const text=await response.text();document.body.replaceChildren(document.createTextNode(text));}
 catch{document.body.replaceChildren(document.createTextNode('Request unavailable. Check normal sign-in before requesting another link.'));}
});</script>`
