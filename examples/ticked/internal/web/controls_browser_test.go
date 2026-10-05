//go:build browser

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/log"
)

// Capture the actual application logger with a finite fixture envelope. The
// browser can inspect redacted observations but cannot manufacture authority.
type browserSecurityLog struct {
	mu       sync.Mutex
	lines    []string
	overflow bool
}

func (l *browserSecurityLog) append(line string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.lines) >= 512 || len(line) > 2048 {
		l.overflow = true

		return
	}

	l.lines = append(l.lines, line)
}
func (l *browserSecurityLog) Debug(v ...any)            { l.append(fmt.Sprint(v...)) }
func (l *browserSecurityLog) Info(v ...any)             { l.append(fmt.Sprint(v...)) }
func (l *browserSecurityLog) Error(v ...any)            { l.append(fmt.Sprint(v...)) }
func (l *browserSecurityLog) Debugf(f string, v ...any) { l.append(fmt.Sprintf(f, v...)) }
func (l *browserSecurityLog) Infof(f string, v ...any)  { l.append(fmt.Sprintf(f, v...)) }
func (l *browserSecurityLog) Errorf(f string, v ...any) { l.append(fmt.Sprintf(f, v...)) }
func (l *browserSecurityLog) With(...any) log.Logger    { return l }

func (l *browserSecurityLog) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lines = nil
	l.overflow = false
}

func (l *browserSecurityLog) events() ([]auth.SecurityEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.overflow {
		return nil, fmt.Errorf("browser capture capacity exceeded")
	}

	events := []auth.SecurityEvent{}

	for _, line := range l.lines {
		if !strings.HasPrefix(line, "Authentication security event: ") {
			continue
		}

		var event auth.SecurityEvent

		err := json.Unmarshal([]byte(strings.TrimPrefix(line, "Authentication security event: ")), &event)
		if err != nil {
			return nil, err
		}

		err = event.Check()
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

func (l *browserSecurityLog) snapshot(w http.ResponseWriter, _ *http.Request) {
	events, err := l.events()
	if err != nil {
		http.Error(w, "Invalid fixture observations", 500)

		return
	}

	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(events)
}

// Proof methods and committed mutations must come from the completed real
// journeys; a valid shape alone cannot establish accepted WebAuthn or recovery.
func (l *browserSecurityLog) verify(t *testing.T, recovery bool) {
	t.Helper()

	events, err := l.events()
	if err != nil {
		t.Fatal(err)
	}

	required := map[auth.SecurityOperation]bool{
		auth.SecurityRegistration: false, auth.SecuritySignin: false,
		auth.SecurityEnrollmentFinish: false, auth.SecurityWebAuthnFinish: false,
		auth.SecurityTOTPSetupFinish: false, auth.SecurityFallbackFinish: false,
		auth.SecurityBackupIssue: false, auth.SecurityFactorRemoval: false,
	}
	if recovery {
		required[auth.SecurityPasswordChange] = false
		required[auth.SecurityPasswordReset] = false
		required[auth.SecurityMailboxVerification] = false
	}

	methods := map[auth.ProofMethod]bool{auth.PasswordProof: false, auth.WebAuthnProof: false, auth.PasswordTOTPProof: false, auth.PasswordBackupProof: false}

	ids := map[string]bool{}
	for _, event := range events {
		if ids[event.ID] {
			t.Fatal("reused security event identifier")
		}

		ids[event.ID] = true
		if event.Outcome == auth.SecurityCommitted || event.Outcome == auth.SecurityAuthenticated {
			if _, needed := required[event.Operation]; needed {
				required[event.Operation] = true
			}
		}

		if event.Outcome == auth.SecurityAuthenticated {
			methods[event.Proof] = true
		}
	}

	for operation, observed := range required {
		if !observed {
			t.Errorf("missing committed browser observation: %s", operation)
		}
	}

	for method, observed := range methods {
		if !observed {
			t.Errorf("missing actual browser proof observation: %d", method)
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, line := range l.lines {
		for _, secret := range []string{"@example.com", "browser password", "distinct safe password", "#token=", "otpauth://", "clientDataJSON", "attestationObject", "users_email_key", "SQLSTATE", "argon2id"} {
			if strings.Contains(line, secret) {
				t.Fatal("browser material leaked to application log")
			}
		}
	}

	t.Logf("checked %d unique redacted terminal browser observations", len(events))
}
