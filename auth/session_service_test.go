// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

type sessionProbe struct {
	*mockQueries
	calls    int
	deadline time.Time
	late     bool
}

func (q *sessionProbe) ValidateSession(ctx context.Context, digest SessionDigest, requirement AccessRequirement, activity SessionActivity, interval time.Duration) (*ValidatedSession, error) {
	q.calls++

	q.deadline, _ = ctx.Deadline()
	if q.late {
		<-ctx.Done()

		return &ValidatedSession{User: &User{ID: "late"}}, nil
	}

	return q.mockQueries.ValidateSession(ctx, digest, requirement, activity, interval)
}

// Failures issue no bearer or stored row; malformed input never reaches storage.
func TestSessionFailure(t *testing.T) {
	queries := &sessionProbe{mockQueries: newMockQueries()}
	svc := newServiceForTest(t, queries, config.New(), log.NewTestLogger("error"))

	_, err := svc.Signup(t.Context(), "entropy@example.com", "a safe distinct password")
	if err != nil {
		t.Fatal(err)
	}

	failure := errors.New("entropy unavailable")
	svc.sessionToken = func() (string, SessionDigest, error) { return "", SessionDigest{}, failure }

	issued, err := testSignin(svc, t.Context(), "entropy@example.com", "a safe distinct password")
	if issued != nil || !errors.Is(err, failure) || len(queries.sessions) != 0 {
		t.Fatal("entropy failure persisted or issued a session")
	}

	for _, activity := range []SessionActivity{NoActivity, SessionActivity(255)} {
		validated, validateErr := svc.ValidateSession(t.Context(), "invalid", testRequirement(), activity)
		if validated != nil || validateErr == nil || queries.calls != 0 {
			t.Fatal("invalid request reached storage")
		}
	}
}

// Virtual time establishes both deadline precedence and late-success rejection.
func TestSessionDeadline(t *testing.T) {
	tests := []struct {
		name   string
		parent time.Duration
		want   time.Duration
	}{
		{"operation", time.Hour, time.Second}, {"earlier parent", 100 * time.Millisecond, 100 * time.Millisecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				queries := &sessionProbe{mockQueries: newMockQueries(), late: true}
				cfg := config.New()
				cfg.Auth.SessionTimeout = "1s"
				svc := newServiceForTest(t, queries, cfg, log.NewTestLogger("error"))

				token, _, err := newSessionToken()
				if err != nil {
					t.Fatal(err)
				}

				now := time.Now()

				ctx, cancel := context.WithTimeout(t.Context(), test.parent)
				defer cancel()

				validated, err := svc.ValidateSession(ctx, token, testRequirement(), NoActivity)
				if validated != nil || !errors.Is(err, context.DeadlineExceeded) || queries.deadline.Sub(now) != test.want {
					t.Fatalf("late success/deadline: %v", err)
				}
			})
		})
	}
}
