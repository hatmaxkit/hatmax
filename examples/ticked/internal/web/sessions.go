// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"errors"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/web"
	"net/http"
	"time"
)

func (h *Handler) renderSessions(w http.ResponseWriter, page *auth.SessionPage, message string) {
	h.tmpl.Render(w, "auth", "sessions", map[string]interface{}{"Title": "Sessions - Ticked", "Page": page, "Error": message})
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

		return
	}

	page, err := h.authSvc.ListSessions(r.Context(), cookie.Value, r.URL.Query().Get("cursor"))
	if errors.Is(err, auth.ErrSessionProofExpired) {
		h.renderSessions(w, nil, "Reauthenticate to manage sessions")

		return
	}

	if err != nil {
		http.Error(w, "Cannot list sessions", http.StatusUnauthorized)

		return
	}

	h.renderSessions(w, page, "")
}

func (h *Handler) handleReauthenticate(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)

		return
	}

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	result, err := h.authSvc.Reauthenticate(r.Context(), cookie.Value, form.String("password"))
	if err != nil {
		h.renderSessions(w, nil, "Reauthentication failed")

		return
	}

	issued, ok := result.CompletedSession()
	if !ok {
		h.renderSessions(w, nil, "Additional authentication is required and unavailable")

		return
	}

	auth.SetSessionCookie(w, issued.Token, int(time.Until(issued.ExpiresAt).Seconds()))
	w.Header().Set("HX-Redirect", "/sessions")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleRevokeSessions(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)

		return
	}

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	scopes := map[string]auth.SessionScope{"current": auth.SessionCurrent, "selected": auth.SessionSelected, "others": auth.SessionOthers, "all": auth.SessionAll}

	selection := auth.SessionSelection{Scope: scopes[form.String("scope")], ID: form.String("id")}

	err = selection.Check()
	if err != nil {
		http.Error(w, "Invalid session selection", http.StatusBadRequest)

		return
	}

	_, err = h.authSvc.RevokeSessions(r.Context(), cookie.Value, selection)
	if err != nil {
		h.renderSessions(w, nil, "Cannot revoke sessions; recent authentication is required")

		return
	}

	destination := "/sessions"

	if selection.Scope == auth.SessionCurrent || selection.Scope == auth.SessionAll {
		auth.ClearSessionCookie(w)

		destination = "/signin"
	}

	w.Header().Set("HX-Redirect", destination)
	w.WriteHeader(http.StatusOK)
}
