// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/middleware"
)

type fallbackService interface {
	BeginTOTPSetup(context.Context, string, string, auth.AccessRequirement) (*auth.TOTPSetup, error)
	ConfirmTOTPSetup(context.Context, string, string, auth.AccessRequirement) error
	BeginFallbackAuthentication(context.Context, string, string, auth.FallbackMethod, auth.AccessRequirement) (*auth.FallbackChallenge, error)
	BeginFallbackStepUp(context.Context, string, string, auth.FallbackMethod, auth.AccessRequirement) (*auth.FallbackChallenge, error)
	FinishFallback(context.Context, string, string, auth.AccessRequirement) (*auth.IssuedSession, error)
	IssueBackupCodes(context.Context, string, auth.AccessRequirement) ([]string, *auth.IssuedSession, error)
}

// FallbackHandler fixes method by route and captures application-owned policies.
// Finish alone sets a committed session cookie; setup confirmation clears it.
type FallbackHandler struct {
	service            fallbackService
	validator          middleware.SessionValidator
	access, management auth.AccessRequirement
}

func NewFallbackHandler(service fallbackService, validator middleware.SessionValidator, access, management auth.AccessRequirement) (*FallbackHandler, error) {
	if service == nil || validator == nil || access.Proof == auth.RequirePhishingResistantMFA || management.Proof == auth.RequirePassword || management.MaxAge == 0 {
		return nil, errors.New("invalid fallback handler configuration")
	}

	for _, r := range []auth.AccessRequirement{access, management} {
		err := r.Check(10 * time.Minute)
		if err != nil {
			return nil, err
		}
	}

	return &FallbackHandler{service: service, validator: validator, access: access, management: management}, nil
}
func (h *FallbackHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireSameOrigin)
		r.Post("/authenticators/totp/setup/begin", h.setup)
		r.Post("/authenticators/totp/setup/finish", h.confirm)
		r.Post("/authenticators/totp/authentication/begin", func(w http.ResponseWriter, r *http.Request) { h.begin(w, r, auth.FallbackTOTP, false) })
		r.Post("/authenticators/backup/authentication/begin", func(w http.ResponseWriter, r *http.Request) { h.begin(w, r, auth.FallbackBackup, false) })
		r.Post("/authenticators/totp/step-up/begin", func(w http.ResponseWriter, r *http.Request) { h.begin(w, r, auth.FallbackTOTP, true) })
		r.Post("/authenticators/backup/step-up/begin", func(w http.ResponseWriter, r *http.Request) { h.begin(w, r, auth.FallbackBackup, true) })
		r.Post("/authenticators/fallback/authentication/finish", func(w http.ResponseWriter, r *http.Request) { h.finish(w, r, auth.FallbackSignin) })
		r.Post("/authenticators/fallback/step-up/finish", func(w http.ResponseWriter, r *http.Request) { h.finish(w, r, auth.FallbackStepUp) })
		r.Post("/authenticators/backup/issue", h.issue)
	})
	router.Get("/authenticators/mfa", h.proof)
}
func fallbackBody(w http.ResponseWriter, r *http.Request, target any) error {
	body, err := readEnrollmentBody(w, r)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	if err != nil {
		return err
	}

	var extra any

	err = decoder.Decode(&extra)
	if err != io.EOF {
		return auth.ErrFallback
	}

	return nil
}

type fallbackCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *FallbackHandler) setup(w http.ResponseWriter, r *http.Request) {
	var input fallbackCredentials

	err := fallbackBody(w, r, &input)
	if err != nil || len(input.Email) > 254 || len(input.Password) > 4096 {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	result, err := h.service.BeginTOTPSetup(r.Context(), input.Email, input.Password, h.access)
	assertionResponse(w, result, err)
}
func (h *FallbackHandler) confirm(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}

	err := fallbackBody(w, r, &input)
	if err != nil || len(input.Code) != 6 {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	err = h.service.ConfirmTOTPSetup(r.Context(), r.Header.Get("X-Fallback-Token"), input.Code, h.access)
	if err == nil {
		auth.ClearSessionCookie(w)
	}

	assertionResponse(w, struct {
		Confirmed bool `json:"confirmed"`
	}{Confirmed: err == nil}, err)
}
func (h *FallbackHandler) begin(w http.ResponseWriter, r *http.Request, method auth.FallbackMethod, stepUp bool) {
	var input fallbackCredentials

	err := fallbackBody(w, r, &input)
	if err != nil || len(input.Email) > 254 || len(input.Password) > 4096 || stepUp && input.Email != "" {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	var result *auth.FallbackChallenge

	if stepUp {
		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil {
			assertionResponse(w, nil, auth.ErrFallback)

			return
		}

		result, err = h.service.BeginFallbackStepUp(r.Context(), cookie.Value, input.Password, method, h.access)
		assertionResponse(w, result, err)

		return
	}

	result, err = h.service.BeginFallbackAuthentication(r.Context(), input.Email, input.Password, method, h.access)
	assertionResponse(w, result, err)
}
func (h *FallbackHandler) finish(w http.ResponseWriter, r *http.Request, purpose auth.FallbackPurpose) {
	token := r.Header.Get("X-Fallback-Token")

	_, parsed, err := auth.ParseFallbackToken(token)
	if err != nil || parsed != purpose {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	var input struct {
		Code string `json:"code"`
	}

	err = fallbackBody(w, r, &input)
	if err != nil || len(input.Code) > 128 {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	issued, err := h.service.FinishFallback(r.Context(), token, input.Code, h.access)
	if err != nil || issued == nil {
		if err == nil {
			err = auth.ErrFallback
		}

		assertionResponse(w, nil, err)

		return
	}

	auth.SetSessionCookie(w, issued.Token, int(time.Until(issued.ExpiresAt).Seconds()))
	assertionResponse(w, issued.Session, nil)
}
func (h *FallbackHandler) issue(w http.ResponseWriter, r *http.Request) {
	var input struct{}

	err := fallbackBody(w, r, &input)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	codes, issued, err := h.service.IssueBackupCodes(r.Context(), cookie.Value, h.management)
	if err != nil || issued == nil {
		if err == nil {
			err = auth.ErrFallback
		}

		assertionResponse(w, nil, err)

		return
	}

	auth.SetSessionCookie(w, issued.Token, int(time.Until(issued.ExpiresAt).Seconds()))
	assertionResponse(w, struct {
		Codes   []string     `json:"codes"`
		Session auth.Session `json:"session"`
	}{Codes: codes, Session: issued.Session}, nil)
}
func (h *FallbackHandler) proof(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		assertionResponse(w, nil, auth.ErrFallback)

		return
	}

	validated, err := h.validator.ValidateSession(r.Context(), cookie.Value, h.access, auth.NoActivity)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	assertionResponse(w, validated.Session, nil)
}
