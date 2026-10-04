// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/middleware"
)

type factorService interface {
	List(context.Context, string, auth.FactorPolicy) ([]auth.Factor, error)
	BeginWebAuthnChange(context.Context, string, auth.FactorSelection, auth.FactorPolicy) (*auth.EnrollmentChallenge, error)
	FinishWebAuthnChange(context.Context, string, string, []byte, auth.FactorPolicy) (*auth.FactorChangeResult, error)
	BeginTOTPChange(context.Context, string, auth.FactorSelection, auth.FactorPolicy) (*auth.TOTPSetup, error)
	FinishTOTPChange(context.Context, string, string, string, auth.FactorPolicy) (*auth.FactorChangeResult, error)
	Remove(context.Context, string, auth.FactorSelection, auth.FactorPolicy) (*auth.FactorChangeResult, error)
}

// FactorHandler captures trusted policy once. Requests select only an owned target.
type FactorHandler struct {
	service factorService
	policy  auth.FactorPolicy
	totp    bool
}

func NewFactorHandler(service factorService, policy auth.FactorPolicy, totp bool) (*FactorHandler, error) {
	if service == nil {
		return nil, errors.New("factor service is required")
	}

	settings := config.EnrollmentSettings{PendingTTL: 10 * time.Minute, RecentProofAge: 10 * time.Minute}

	err := policy.Check(settings)
	if err != nil {
		return nil, err
	}

	return &FactorHandler{service: service, policy: policy, totp: totp}, nil
}
func (h *FactorHandler) RegisterRoutes(router chi.Router) {
	router.Get("/authenticators/manage", h.page)
	router.Get("/authenticators/factors", h.list)
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireSameOrigin)
		r.Post("/authenticators/factors/remove", h.remove)
		r.Post("/authenticators/factors/webauthn/begin", h.beginWebAuthn)
		r.Post("/authenticators/factors/webauthn/finish", h.finishWebAuthn)

		if h.totp {
			r.Post("/authenticators/factors/totp/begin", h.beginTOTP)
			r.Post("/authenticators/factors/totp/finish", h.finishTOTP)
		}
	})
}
func factorCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		return "", auth.ErrFactorChange
	}

	return cookie.Value, nil
}
func factorResponse(w http.ResponseWriter, result *auth.FactorChangeResult, err error) {
	if err == nil && result == nil {
		err = auth.ErrFactorChange
	}

	if err == nil {
		if result.Issued == nil {
			auth.ClearSessionCookie(w)
		} else {
			auth.SetSessionCookie(w, result.Issued.Token, int(time.Until(result.Issued.ExpiresAt).Seconds()))
		}
	}

	assertionResponse(w, result, err)
}
func (h *FactorHandler) list(w http.ResponseWriter, r *http.Request) {
	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	factors, err := h.service.List(r.Context(), token, h.policy)
	assertionResponse(w, factors, err)
}
func factorTarget(w http.ResponseWriter, r *http.Request) (auth.FactorSelection, error) {
	var input struct {
		Target auth.FactorSelection `json:"target"`
	}

	err := fallbackBody(w, r, &input)

	return input.Target, err
}
func (h *FactorHandler) remove(w http.ResponseWriter, r *http.Request) {
	target, err := factorTarget(w, r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	result, err := h.service.Remove(r.Context(), token, target, h.policy)
	factorResponse(w, result, err)
}
func (h *FactorHandler) beginWebAuthn(w http.ResponseWriter, r *http.Request) {
	target, err := factorTarget(w, r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	result, err := h.service.BeginWebAuthnChange(r.Context(), token, target, h.policy)
	assertionResponse(w, result, err)
}
func (h *FactorHandler) finishWebAuthn(w http.ResponseWriter, r *http.Request) {
	body, err := readEnrollmentBody(w, r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	result, err := h.service.FinishWebAuthnChange(r.Context(), token, r.Header.Get("X-Factor-Change-Token"), body, h.policy)
	factorResponse(w, result, err)
}
func (h *FactorHandler) beginTOTP(w http.ResponseWriter, r *http.Request) {
	target, err := factorTarget(w, r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	result, err := h.service.BeginTOTPChange(r.Context(), token, target, h.policy)
	assertionResponse(w, result, err)
}
func (h *FactorHandler) finishTOTP(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}

	err := fallbackBody(w, r, &input)
	if err != nil || len(input.Code) != 6 {
		assertionResponse(w, nil, auth.ErrFactorChange)

		return
	}

	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	result, err := h.service.FinishTOTPChange(r.Context(), token, r.Header.Get("X-Factor-Change-Token"), input.Code, h.policy)
	factorResponse(w, result, err)
}

// The minimal management view uses production JSON endpoints and real browser
// registration. It displays only safe metadata and transient setup/code results.
var factorPage = template.Must(template.New("factors").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Authenticators</title></head><body>
<h1>Authenticators</h1><p>Changes require recent management proof. Removing or replacing the factor used by this session signs you out. The last usable factor cannot be removed.</p>
<ul>{{range .Factors}}<li>{{if eq .Kind 1}}Passkey{{else}}TOTP{{end}} · {{.ID}} <button data-kind="{{.Kind}}" data-id="{{.ID}}" data-revision="{{.Revision}}" data-action="remove">Remove</button> <button data-kind="{{.Kind}}" data-id="{{.ID}}" data-revision="{{.Revision}}" data-action="webauthn">Replace with passkey</button> {{if $.Fallback}}<button data-kind="{{.Kind}}" data-id="{{.ID}}" data-revision="{{.Revision}}" data-action="totp">Replace with TOTP</button>{{end}}</li>{{end}}</ul>
<button data-action="webauthn">Add passkey</button>{{if .Fallback}}<button data-action="totp">Add TOTP</button><button data-action="backup">Regenerate backup codes</button><p><label>Confirmation code <input id="code" maxlength="6" inputmode="numeric"></label> <button id="confirm">Confirm TOTP</button></p>{{end}}<pre id="result"></pre>
<script>
const output=document.querySelector('#result');let pending='';
const bytes=s=>Uint8Array.from(atob(s.replace(/-/g,'+').replace(/_/g,'/')),c=>c.charCodeAt(0));
const encode=b=>btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'');
async function post(path,body,token){const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json',...(token?{'X-Factor-Change-Token':token}:{})},body:JSON.stringify(body)});if(!r.ok)throw Error(await r.text());return r.json()}
document.querySelectorAll('[data-action]').forEach(b=>b.onclick=async()=>{try{const target=b.dataset.id?{kind:Number(b.dataset.kind),id:b.dataset.id,revision:Number(b.dataset.revision)}:undefined;const action=b.dataset.action;if(action==='remove'){await post('/authenticators/factors/remove',{target});location.reload();return}if(action==='backup'){const result=await post('/authenticators/backup/issue',{});output.textContent=result.codes.join('\n');return}const begin=await post('/authenticators/factors/'+action+'/begin',{target});if(action==='totp'){pending=begin.token;output.textContent=begin.url;return}const options=begin.options.publicKey;options.challenge=bytes(options.challenge);options.user.id=bytes(options.user.id);if(options.excludeCredentials)options.excludeCredentials.forEach(c=>c.id=bytes(c.id));const credential=await navigator.credentials.create({publicKey:options});await post('/authenticators/factors/webauthn/finish',{id:credential.id,rawId:encode(credential.rawId),type:credential.type,response:{clientDataJSON:encode(credential.response.clientDataJSON),attestationObject:encode(credential.response.attestationObject),transports:credential.response.getTransports()},clientExtensionResults:credential.getClientExtensionResults()},begin.token);location.reload()}catch(e){output.textContent=e.message}});
const confirm=document.querySelector('#confirm');if(confirm)confirm.onclick=async()=>{try{await post('/authenticators/factors/totp/finish',{code:document.querySelector('#code').value},pending);pending='';location.reload()}catch(e){output.textContent=e.message}};
</script></body></html>`))

func (h *FactorHandler) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	token, err := factorCookie(r)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	factors, err := h.service.List(r.Context(), token, h.policy)
	if err != nil {
		assertionResponse(w, nil, err)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = factorPage.Execute(w, struct {
		Factors  []auth.Factor
		Fallback bool
	}{factors, h.totp})
}
