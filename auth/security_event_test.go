// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

type panickingSecurityError struct{}

func (panickingSecurityError) Error() string { return "private callback detail" }
func (panickingSecurityError) Is(error) bool { panic("private callback detail") }

type securityObserverFunc func(context.Context, SecurityEvent) error

func (f securityObserverFunc) Observe(ctx context.Context, e SecurityEvent) error { return f(ctx, e) }

func observationForTest(t *testing.T, observer SecurityObserver, cfg config.SecurityObservationConfig) *SecurityObservations {
	t.Helper()

	s, err := NewSecurityObservations(observer, cfg)
	if err != nil {
		t.Fatal(err)
	}

	return s
}
func securityTestEvent() SecurityEvent {
	return SecurityEvent{ID: model.NewID(), OccurredAt: time.Now().UTC(), Operation: SecuritySignin, Outcome: SecurityInvalidProof}
}

// Missing dependencies fail before any authentication work can bypass observations.
func TestSecurityConstruction(t *testing.T) {
	_, err := NewSecurityObservations(nil, config.SecurityObservationConfig{})
	if err == nil {
		t.Fatal("nil observer accepted")
	}

	for _, observations := range []*SecurityObservations{nil, {}} {
		_, err = NewService(newMockQueries(), config.New(), passwordCheckerFunc(func(context.Context, string) (bool, error) { return false, nil }), newAdmissionForTest(t), observations, log.NewTestLogger("error"))
		if err == nil {
			t.Fatal("uninitialized observations accepted")
		}
	}
}

// Observer failures are fixed diagnostics and cannot expose arbitrary error text.
func TestSecurityDelivery(t *testing.T) {
	for _, tc := range []struct {
		name     string
		observer securityObserverFunc
		expected SecurityObservationDiagnostics
	}{
		{"delivered", func(context.Context, SecurityEvent) error { return nil }, SecurityObservationDiagnostics{Delivered: 1}},
		{"rejected", func(context.Context, SecurityEvent) error { return ErrSecurityObservationRejected }, SecurityObservationDiagnostics{Rejected: 1}},
		{"operating", func(context.Context, SecurityEvent) error { return errors.New("secret adapter detail") }, SecurityObservationDiagnostics{Operating: 1}},
		{"panic", func(context.Context, SecurityEvent) error { panic("secret adapter detail") }, SecurityObservationDiagnostics{Operating: 1}},
		{"error classification panic", func(context.Context, SecurityEvent) error { return panickingSecurityError{} }, SecurityObservationDiagnostics{Operating: 1}},
		{"late success", func(ctx context.Context, _ SecurityEvent) error {
			<-ctx.Done()

			return nil
		}, SecurityObservationDiagnostics{Deadline: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := observationForTest(t, tc.observer, config.SecurityObservationConfig{Timeout: "5ms"})
			s.observe(t.Context(), securityTestEvent())

			if got := s.Diagnostics(); got != tc.expected {
				t.Fatalf("diagnostics: %+v", got)
			}
		})
	}

	t.Run("canceled caller", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		s := observationForTest(t, securityObserverFunc(func(context.Context, SecurityEvent) error {
			t.Fatal("canceled callback invoked")

			return nil
		}), config.SecurityObservationConfig{})
		s.observe(ctx, securityTestEvent())

		if s.Diagnostics().Canceled != 1 {
			t.Fatal("cancellation not classified")
		}
	})
	t.Run("shorter caller deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		deadline, _ := ctx.Deadline()
		s := observationForTest(t, securityObserverFunc(func(ctx context.Context, _ SecurityEvent) error {
			observed, ok := ctx.Deadline()
			if !ok || !observed.Before(deadline) {
				t.Fatal("caller deadline extended")
			}

			return nil
		}), config.SecurityObservationConfig{})
		s.observe(ctx, securityTestEvent())

		if s.Diagnostics().Delivered != 1 {
			t.Fatal("timely observer failed")
		}
	})
}

// Saturation is non-waiting; re-entry cannot deadlock while holding callback capacity.
func TestSecurityCapacity(t *testing.T) {
	t.Run("concurrent saturation", func(t *testing.T) {
		started := make(chan struct{}, 2)
		release := make(chan struct{})

		var calls atomic.Int32

		s := observationForTest(t, securityObserverFunc(func(ctx context.Context, _ SecurityEvent) error {
			calls.Add(1)

			started <- struct{}{}

			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}), config.SecurityObservationConfig{})

		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() { s.observe(t.Context(), securityTestEvent()) })
		}

		for range 2 {
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("callbacks did not start")
			}
		}

		s.observe(t.Context(), securityTestEvent())

		if calls.Load() != 2 || s.Diagnostics().Saturated != 1 {
			t.Fatal("saturated callback invoked or queued")
		}

		close(release)
		wg.Wait()

		if s.Diagnostics().Delivered != 2 {
			t.Fatal("accepted callbacks failed")
		}
	})
	t.Run("reentry", func(t *testing.T) {
		svc := newServiceForTest(t, newMockQueries(), config.New(), log.NewTestLogger("error"))
		s := observationForTest(t, securityObserverFunc(func(ctx context.Context, _ SecurityEvent) error {
			err := svc.Signout(ctx, "not a bearer")
			if !errors.Is(err, ErrSessionToken) {
				t.Fatal("reentry changed authentication error")
			}

			return nil
		}), config.SecurityObservationConfig{Concurrency: 1})
		svc.observations = s

		err := svc.Signout(t.Context(), "not a bearer")
		if !errors.Is(err, ErrSessionToken) || s.Diagnostics().Saturated != 1 || s.Diagnostics().Delivered != 1 {
			t.Fatal("reentry waited or changed authority")
		}
	})
}

// Arbitrary untrusted strings cannot expand or inject the bounded event shape.
func FuzzSecurityEvent(f *testing.F) {
	for _, value := range []string{"subject-1", "raw@example.com", "\nsecret", "https://example.com/token", strings.Repeat("a", 129), "\xff"} {
		f.Add(value, value, uint8(0))
	}

	f.Fuzz(func(t *testing.T, subject, record string, proof uint8) {
		e := securityTestEvent()
		e.Subject = securityReference(subject)

		e.Record = securityReference(record)
		if proof >= uint8(PasswordProof) && proof <= uint8(PasswordBackupProof) {
			e.Proof = ProofMethod(proof)
		}

		err := e.Check()
		if err != nil {
			t.Fatal(err)
		}

		encoded, err := json.Marshal(e)
		if err != nil || len(encoded) > 1024 {
			t.Fatal("event exceeded envelope")
		}

		for _, value := range []string{e.Subject, e.Record} {
			if value != "" && (len(value) > 128 || strings.ContainsAny(value, "@:/ \r\n\x00")) {
				t.Fatal("unsafe reference survived")
			}
		}
	})
}

// The closed shape rejects injected references, unknown enums and fabricated completion.
func TestSecurityShape(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*SecurityEvent)
	}{
		{"operation", func(e *SecurityEvent) { e.Operation = "request-selected" }},
		{"outcome", func(e *SecurityEvent) { e.Outcome = "request-selected" }},
		{"identity", func(e *SecurityEvent) { e.Subject = "raw@example.com" }},
		{"long reference", func(e *SecurityEvent) { e.Record = strings.Repeat("a", 129) }},
		{"newline", func(e *SecurityEvent) { e.Record = "safe\nprivate" }},
		{"proof method", func(e *SecurityEvent) { e.Proof = ProofMethod(5) }},
		{"unverified completion", func(e *SecurityEvent) { e.Outcome = SecurityAuthenticated }},
		{"time", func(e *SecurityEvent) { e.OccurredAt = time.Time{} }},
		{"timezone", func(e *SecurityEvent) { e.OccurredAt = e.OccurredAt.In(time.FixedZone("untrusted", 3600)) }},
		{"bearer id", func(e *SecurityEvent) { e.ID = "token-derived" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := securityTestEvent()
			tc.change(&event)

			err := event.Check()
			if !errors.Is(err, ErrSecurityEvent) {
				t.Fatal("unsafe shape accepted")
			}

			var calls int

			s := observationForTest(t, securityObserverFunc(func(context.Context, SecurityEvent) error {
				calls++

				return nil
			}), config.SecurityObservationConfig{})
			s.observe(t.Context(), event)

			if calls != 0 || s.Diagnostics().Operating != 1 {
				t.Fatal("invalid event reached observer")
			}
		})
	}
}

// Setup flags select explanatory pending state, never verified MFA or a session.
func TestSecurityPendingOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		required RequiredProof
		enrolled bool
		expected SecurityOutcome
	}{
		{"enrollment", RequireMFA, false, SecurityPendingEnrollment},
		{"factor proof", RequireMFA, true, SecurityPendingProof},
		{"unavailable method", RequirePhishingResistantMFA, true, SecurityDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := newMockQueries()
			svc := newServiceForTest(t, q, config.New(), log.NewTestLogger("error"))

			user, err := svc.Signup(t.Context(), "session@example.com", "a distinct safe password")
			if err != nil {
				t.Fatal(err)
			}

			q.users[user.ID].TOTPEnabled = tc.enrolled

			var events []SecurityEvent

			svc.observations = observationForTest(t, securityObserverFunc(func(_ context.Context, event SecurityEvent) error {
				events = append(events, event)

				return nil
			}), config.SecurityObservationConfig{})
			required := testRequirement()
			required.Proof = tc.required

			result, err := svc.Signin(t.Context(), user.Email, "a distinct safe password", required)
			if err != nil || result.Issued != nil || len(q.sessions) != 0 {
				t.Fatal("pending password proof authorized access")
			}

			if len(events) != 1 || events[0].Outcome != tc.expected || events[0].Proof != PasswordProof {
				t.Fatal("pending event manufactured MFA or duplicated")
			}
		})
	}
}
