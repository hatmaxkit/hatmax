//go:build integration || browser

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"testing"
)

// Shared explicit fixture material proves real admission without process fallback.
func webCredentialAdmission(t *testing.T, q *authfeat.Queries, cfg config.CredentialAdmissionConfig) *auth.CredentialAdmission {
	t.Helper()
	store, err := authfeat.NewCredentialAdmissionStore(q, "web-fixture", make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	admission, err := auth.NewCredentialAdmission(store, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return admission
}
