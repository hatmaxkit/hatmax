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

type webAuthnService interface {
	BeginWebAuthnAuthentication(context.Context, string, auth.AccessRequirement) (*auth.AssertionChallenge, error)
	BeginWebAuthnStepUp(context.Context, string, auth.AccessRequirement) (*auth.AssertionChallenge, error)
	FinishWebAuthn(context.Context, string, []byte, auth.AccessRequirement) (*auth.IssuedSession, error)
}

// WebAuthnHandler captures trusted policy for begin/finish and a strong route.
type WebAuthnHandler struct {
	service     webAuthnService
	validator   middleware.SessionValidator
	requirement auth.AccessRequirement
}

func NewWebAuthnHandler(service webAuthnService, validator middleware.SessionValidator, requirement auth.AccessRequirement) (*WebAuthnHandler, error) {
	if service == nil || validator == nil {
		return nil, errors.New("WebAuthn dependencies are required")
	}

	err := requirement.Check(10 * time.Minute)
	if err != nil {
		return nil, err
	}

	return &WebAuthnHandler{service: service, validator: validator, requirement: requirement}, nil
}
func (h *WebAuthnHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireSameOrigin)
		r.Post("/authenticators/authentication/begin", h.begin)
		r.Post("/authenticators/step-up/begin", h.stepUp)
		r.Post("/authenticators/authentication/finish", h.finishSignin)
		r.Post("/authenticators/step-up/finish", h.finishStepUp)
	})
	router.Get("/authenticators/proof", h.proof)
}
func assertionResponse(w http.ResponseWriter, result any, err error) {
	w.Header().Set("Cache-Control", "no-store")

	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, auth.ErrEnrollmentBusy) || errors.Is(err, auth.ErrEnrollmentAttempts) || errors.Is(err, auth.ErrEnrollmentCapacity) || errors.Is(err, auth.ErrSessionCapacity) {
			status = http.StatusTooManyRequests
		}

		http.Error(w, "Authentication unavailable", status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
func (h *WebAuthnHandler) begin(w http.ResponseWriter, r *http.Request) {
	body, err := readEnrollmentBody(w, r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	var input struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&input)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	var extra any

	err = decoder.Decode(&extra)
	if err != io.EOF || len(input.Email) == 0 || len(input.Email) > 254 {
		assertionResponse(w, nil, auth.ErrWebAuthn)

		return
	}

	result, err := h.service.BeginWebAuthnAuthentication(r.Context(), input.Email, h.requirement)
	assertionResponse(w, result, err)
}
func (h *WebAuthnHandler) stepUp(w http.ResponseWriter, r *http.Request) {
	body, err := readEnrollmentBody(w, r)
	if err != nil || string(bytes.TrimSpace(body)) != "{}" {
		assertionResponse(w, nil, auth.ErrWebAuthn)

		return
	}

	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		assertionResponse(w, nil, auth.ErrWebAuthn)

		return
	}

	result, err := h.service.BeginWebAuthnStepUp(r.Context(), cookie.Value, h.requirement)
	assertionResponse(w, result, err)
}
func (h *WebAuthnHandler) finishSignin(w http.ResponseWriter, r *http.Request) {
	h.finish(w, r, "assert1.")
}
func (h *WebAuthnHandler) finishStepUp(w http.ResponseWriter, r *http.Request) {
	h.finish(w, r, "stepup1.")
}
func (h *WebAuthnHandler) finish(w http.ResponseWriter, r *http.Request, prefix string) {
	token := r.Header.Get("X-Assertion-Token")
	if len(token) != 51 || token[:8] != prefix {
		assertionResponse(w, nil, auth.ErrWebAuthn)

		return
	}

	body, err := readEnrollmentBody(w, r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	issued, err := h.service.FinishWebAuthn(r.Context(), token, body, h.requirement)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	if issued == nil {
		assertionResponse(w, nil, auth.ErrWebAuthn)

		return
	}

	auth.SetSessionCookie(w, issued.Token, int(time.Until(issued.ExpiresAt).Seconds()))
	assertionResponse(w, issued.Session, nil)
}

func (h *WebAuthnHandler) proof(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		assertionResponse(w, nil, auth.ErrWebAuthn)

		return
	}

	validated, err := h.validator.ValidateSession(r.Context(), cookie.Value, h.requirement, auth.NoActivity)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	assertionResponse(w, validated.Session, nil)
}
