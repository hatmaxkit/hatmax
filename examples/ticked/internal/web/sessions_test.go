// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"errors"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/ui"
	"hatmax.adrianpk.com/web"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func controlHandler(t *testing.T, fake *fakeAuthSvc) *Handler {
	t.Helper()

	logger := log.NewTestLogger("error")

	templates := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))

	err := templates.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return &Handler{tmpl: templates, log: logger, authSvc: fake}
}
func controlRequest(path string, form url.Values) *http.Request {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "old-bearer"})

	return request
}

// Failed or non-authorizing reauthentication cannot replace the browser bearer.
func TestReauthenticationCookie(t *testing.T) {
	cases := []struct {
		name    string
		outcome auth.AuthenticationOutcome
		reason  auth.AuthenticationReason
		err     error
		cookie  bool
	}{
		{"completed", auth.AuthenticationCompleted, auth.AuthenticationSatisfied, nil, true},
		{"enrollment", auth.AuthenticationPendingEnrollment, auth.AuthenticationMethodUnavailable, nil, false},
		{"proof", auth.AuthenticationPendingProof, auth.AuthenticationMethodUnavailable, nil, false},
		{"denied", auth.AuthenticationDenied, auth.AuthenticationMethodUnavailable, nil, false},
		{"contradictory", auth.AuthenticationCompleted, auth.AuthenticationMethodUnavailable, nil, false},
		{"storage failure", auth.AuthenticationCompleted, auth.AuthenticationSatisfied, errors.New("private database diagnostic"), false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeAuthSvc{result: &auth.AuthenticationResult{Outcome: test.outcome, Reason: test.reason, Issued: &auth.IssuedSession{Token: "replacement", Session: auth.Session{ExpiresAt: time.Now().Add(time.Hour)}}}, signinErr: test.err}
			h := controlHandler(t, fake)
			response := httptest.NewRecorder()
			h.handleReauthenticate(response, controlRequest("/reauthenticate", url.Values{"password": {"secret input"}, "subject": {"foreign"}, "proof": {"password"}, "policy_revision": {"client-selected"}}))

			cookies := response.Result().Cookies()
			if (len(cookies) == 1) != test.cookie || (response.Header().Get("HX-Redirect") == "/sessions") != test.cookie || strings.Contains(response.Body.String(), "private database diagnostic") {
				t.Fatal("unauthorized cookie or diagnostic")
			}

			if test.cookie && (cookies[0].Value != "replacement" || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode) {
				t.Fatal("replacement cookie contract")
			}

			if fake.reauthToken != "old-bearer" || fake.reauthPassword != "secret input" {
				t.Fatal("reauthentication input")
			}
		})
	}
}

// Management errors never clear the cookie or report a successful deletion.
func TestSessionControlTransport(t *testing.T) {
	for _, test := range []struct {
		name, scope string
		err         error
		clear       bool
	}{{"all", "all", nil, true}, {"others", "others", nil, false}, {"current", "current", nil, true}, {"stale", "all", auth.ErrSessionProofExpired, false}, {"rollback", "all", errors.New("private database diagnostic"), false}} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeAuthSvc{signoutErr: test.err, revokeCount: 1}
			response := httptest.NewRecorder()
			controlHandler(t, fake).handleRevokeSessions(response, controlRequest("/sessions/revoke", url.Values{"scope": {test.scope}, "subject": {"foreign"}}))

			if (len(response.Result().Cookies()) == 1) != test.clear || strings.Contains(response.Body.String(), "private database diagnostic") {
				t.Fatal("deletion transport")
			}

			if test.err != nil && response.Header().Get("HX-Redirect") != "" {
				t.Fatal("false deletion success")
			}

			if fake.selection.ID != "" {
				t.Fatal("request subject selected a target")
			}
		})
	}
}

// Parse and render the production template with current metadata and paging.
func TestSessionScreen(t *testing.T) {
	tmpl, err := template.New("sessions.html").Funcs(ui.FuncMap()).ParseFiles("../../assets/templates/auth/sessions.html")
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()

	err = tmpl.Execute(response, map[string]interface{}{"Title": "Sessions", "Page": &auth.SessionPage{CurrentID: "current", Sessions: []auth.Session{{ID: "current"}, {ID: "other"}}, NextCursor: "bmV4dA"}})
	if err != nil || !strings.Contains(response.Body.String(), "(current)") || !strings.Contains(response.Body.String(), "/sessions?cursor=bmV4dA") {
		t.Fatal("session template did not render current/paging state")
	}
}

// Failed withdrawal cannot claim that the durable bearer was revoked.
func TestSignoutFailure(t *testing.T) {
	fake := &fakeAuthSvc{signoutErr: errors.New("storage unavailable")}
	response := httptest.NewRecorder()
	controlHandler(t, fake).handleSignout(response, controlRequest("/signout", nil))

	if response.Code != http.StatusServiceUnavailable || len(response.Result().Cookies()) != 0 || response.Header().Get("HX-Redirect") != "" {
		t.Fatal("failed signout reported success")
	}
}
