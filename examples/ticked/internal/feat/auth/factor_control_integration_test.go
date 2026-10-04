//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"net/url"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
)

func factorConfig() config.AuthenticatorConfig {
	return config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}}
}
func factorService(t *testing.T, f webAuthnFixture, q core.FactorQueries) *core.FactorService {
	t.Helper()
	fallback := fallbackService(t, f.base, f.q, config.FallbackConfig{})

	svc, err := core.NewFactorService(f.base, q, f.svc.AuthenticatorService, fallback)
	if err != nil {
		t.Fatal(err)
	}

	return svc
}
func factorPolicy() core.FactorPolicy {
	return core.FactorPolicy{Management: webAuthnRequirement(), Access: core.RequirePhishingResistantMFA}
}
func factorSelection(f *core.Authenticator) core.FactorSelection {
	return core.FactorSelection{Kind: core.FactorWebAuthn, ID: f.ID, Revision: 1}
}
func addPasskey(t *testing.T, f webAuthnFixture, svc *core.FactorService, actor *core.IssuedSession, target core.FactorSelection, policy core.FactorPolicy) (*core.FactorChangeResult, webAuthnFixture) {
	t.Helper()

	begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, target, policy)
	if err != nil {
		t.Fatal(err)
	}

	body, key, id := enrollmentKeyResponseFlags(t, begin, 0x45)

	result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
	if err != nil || result == nil || result.Factor == nil {
		t.Fatalf("change completion: %v", err)
	}

	next := f
	next.key = key
	next.id = id
	next.factor = &core.Authenticator{ID: result.Factor.ID, CreatedAt: result.Factor.CreatedAt}

	return result, next
}

// Actual production verification and PostgreSQL establish additional enrollment,
// atomic replacement/removal, last-factor protection and retained proof coherence.
func TestFactorChanges(t *testing.T) {
	t.Run("WebAuthn lifecycle", testFactorWebAuthn)
	t.Run("TOTP lifecycle", testFactorTOTP)
	t.Run("TOTP upgrade", testFactorUpgrade)
}
func testFactorWebAuthn(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	svc := factorService(t, f, f.q)
	policy := factorPolicy()
	actor := finishAssertion(t, f, beginAssertion(t, f), 1)

	factors, err := svc.List(t.Context(), actor.Token, policy)
	if err != nil || len(factors) != 1 || factors[0].ID != f.factor.ID {
		t.Fatalf("own list: %v", err)
	}

	_, err = svc.Remove(t.Context(), actor.Token, factorSelection(f.factor), policy)
	if err == nil {
		t.Fatal("removed last usable factor")
	}

	other := finishAssertion(t, f, beginAssertion(t, f), 2)
	pending := beginAssertion(t, f)

	result, next := addPasskey(t, f, svc, actor, core.FactorSelection{}, policy)
	if result.Issued == nil || result.Issued.Proof != actor.Proof || result.Issued.Generation != actor.Generation+1 || result.Issued.AuthVersion != actor.AuthVersion+1 || !result.Issued.ExpiresAt.Equal(actor.ExpiresAt) {
		t.Fatal("retained actor was not coherently rotated")
	}

	for _, old := range []string{actor.Token, other.Token} {
		_, err = f.base.ValidateSession(t.Context(), old, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("old session survived mutation")
		}
	}

	issued, err := f.svc.FinishWebAuthn(t.Context(), pending.Token, signedAssertion(t, f, pending, 3, 5, nil), webAuthnRequirement())
	if err == nil || issued != nil {
		t.Fatal("old pending survived mutation")
	}
	// Removing the actor's constituent signs it out even while another usable
	// primary factor remains. Its replacement must prove itself on a fresh sign-in.
	removed, err := svc.Remove(t.Context(), result.Issued.Token, factorSelection(f.factor), policy)
	if err != nil || removed.Issued != nil {
		t.Fatalf("removed proof retained: %v", err)
	}

	actual := finishAssertion(t, next, beginAssertion(t, next), 1)

	replacement, replaced := addPasskey(t, next, svc, actual, factorSelection(next.factor), policy)
	if replacement.Issued != nil {
		t.Fatal("replacement retained proof of removed factor")
	}

	_ = finishAssertion(t, replaced, beginAssertion(t, replaced), 1)

	var count int

	err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM authenticators").Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("replacement cardinality: %d %v", count, err)
	}
}
func testFactorTOTP(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	svc := factorService(t, f, f.q)
	policy := factorPolicy()
	actor := finishAssertion(t, f, beginAssertion(t, f), 1)

	setup, err := svc.BeginTOTPChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(setup.URL)
	if err != nil {
		t.Fatal(err)
	}

	code, err := previousTOTPCode(t, parsed.Query().Get("secret"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.FinishTOTPChange(t.Context(), actor.Token, setup.Token, code, policy)
	if err != nil || result.Issued == nil || result.Factor.Kind != core.FactorTOTP {
		t.Fatalf("TOTP addition: %v", err)
	}

	_, err = svc.BeginTOTPChange(t.Context(), result.Issued.Token, core.FactorSelection{}, policy)
	if err == nil {
		t.Fatal("second TOTP admitted")
	}

	_, err = svc.Remove(t.Context(), result.Issued.Token, factorSelection(f.factor), policy)
	if err == nil {
		t.Fatal("removed last phishing-resistant factor")
	}
	// Replacing a non-actor TOTP retains actual WebAuthn proof unchanged.
	next, err := svc.BeginTOTPChange(t.Context(), result.Issued.Token, result.Factor.FactorSelection, policy)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err = url.Parse(next.URL)
	if err != nil {
		t.Fatal(err)
	}

	code, err = totp.GenerateCode(parsed.Query().Get("secret"), time.Now())
	if err != nil {
		t.Fatal(err)
	}

	replaced, err := svc.FinishTOTPChange(t.Context(), result.Issued.Token, next.Token, code, policy)
	if err != nil || replaced.Issued == nil || replaced.Issued.Proof != actor.Proof || replaced.Factor.ID == result.Factor.ID {
		t.Fatalf("TOTP replacement: %v", err)
	}

	removed, err := svc.Remove(t.Context(), replaced.Issued.Token, replaced.Factor.FactorSelection, policy)
	if err != nil || removed.Issued == nil {
		t.Fatalf("TOTP removal: %v", err)
	}
}
func testFactorUpgrade(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{})

	enrollment, err := core.NewAuthenticatorService(f.base, f.q, factorConfig())
	if err != nil {
		t.Fatal(err)
	}

	svc, err := core.NewFactorService(f.base, f.q, enrollment, f.svc)
	if err != nil {
		t.Fatal(err)
	}

	actor := finishTOTP(t, f)

	_, err = svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, factorPolicy())
	if err == nil {
		t.Fatal("MFA weakened phishing-resistant management")
	}

	lower := core.FactorPolicy{Management: fallbackRequirement(), Access: core.RequireMFA}

	begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, lower)
	if err != nil {
		t.Fatal(err)
	}

	body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)

	result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, lower)
	if err != nil || result.Issued == nil || result.Issued.Proof != actor.Proof {
		t.Fatalf("allowed TOTP upgrade: %v", err)
	}

	_, err = f.base.ValidateSession(t.Context(), result.Issued.Token, webAuthnRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("registration manufactured phishing resistance")
	}
}
