// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"testing"
)

// securityObservationsForTest explicitly discards observations for existing unrelated fixtures.
func securityObservationsForTest(t *testing.T) *core.SecurityObservations {
	t.Helper()

	observations, err := core.NewSecurityObservations(core.DiscardSecurityObserver{}, config.SecurityObservationConfig{})
	if err != nil {
		t.Fatal(err)
	}

	return observations
}
