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

type enrollmentService interface {
	BeginWebAuthnEnrollment(context.Context, string, string, auth.AccessRequirement) (*auth.EnrollmentChallenge, error)
	FinishWebAuthnEnrollment(context.Context, string, []byte, auth.AccessRequirement) (*auth.Authenticator, error)
}

// EnrollmentHandler exposes restricted JSON begin/finish endpoints. Trusted
// policy is captured at construction, never selected through a request field.
type EnrollmentHandler struct {
	service     enrollmentService
	requirement auth.AccessRequirement
}

func NewEnrollmentHandler(service enrollmentService, requirement auth.AccessRequirement) (*EnrollmentHandler, error) {
	if service == nil {
		return nil, errors.New("enrollment service is required")
	}

	err := requirement.Check(10 * time.Minute)
	if err != nil {
		return nil, err
	}

	return &EnrollmentHandler{service: service, requirement: requirement}, nil
}

func (h *EnrollmentHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireSameOrigin)
		r.Post("/authenticators/enrollment/begin", h.begin)
		r.Post("/authenticators/enrollment/finish", h.finish)
	})
}

func readEnrollmentBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, auth.ErrEnrollment
	}

	r.Body = http.MaxBytesReader(w, r.Body, auth.MaxEnrollmentBody)

	return io.ReadAll(r.Body)
}

func enrollmentResponse(w http.ResponseWriter, result any, err error) {
	w.Header().Set("Cache-Control", "no-store")

	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, auth.ErrEnrollmentCapacity) || errors.Is(err, auth.ErrEnrollmentBusy) || errors.Is(err, auth.ErrEnrollmentAttempts) {
			status = http.StatusTooManyRequests
		}

		http.Error(w, "Enrollment unavailable", status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *EnrollmentHandler) begin(w http.ResponseWriter, r *http.Request) {
	body, err := readEnrollmentBody(w, r)
	if err != nil {
		enrollmentResponse(w, nil, err)

		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&input)
	if err != nil {
		enrollmentResponse(w, nil, err)

		return
	}

	if auth.CheckCredentialIdentity(input.Email) != nil || input.Password == "" || len(input.Password) > 4096 {
		enrollmentResponse(w, nil, auth.ErrEnrollment)

		return
	}

	var extra any

	err = decoder.Decode(&extra)
	if err != io.EOF {
		enrollmentResponse(w, nil, auth.ErrEnrollment)

		return
	}

	ctx, cancel, admitted := publicAuthenticationWork(w, r)
	if !admitted {
		return
	}
	defer cancel()

	result, err := h.service.BeginWebAuthnEnrollment(ctx, input.Email, input.Password, h.requirement)
	if err != nil {
		publicPasswordDenial(w, r)

		return
	}

	enrollmentResponse(w, result, nil)
}

func (h *EnrollmentHandler) finish(w http.ResponseWriter, r *http.Request) {
	body, err := readEnrollmentBody(w, r)
	if err != nil {
		enrollmentResponse(w, nil, err)

		return
	}

	result, err := h.service.FinishWebAuthnEnrollment(r.Context(), r.Header.Get("X-Enrollment-Token"), body, h.requirement)
	if err == nil {
		auth.ClearSessionCookie(w)
	}

	enrollmentResponse(w, result, err)
}
