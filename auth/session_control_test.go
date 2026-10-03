// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"math"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// Closed scopes and bounded canonical cursors reject malformed input without IO.
func TestSessionSelection(t *testing.T) {
	cases := []struct {
		name      string
		selection SessionSelection
		valid     bool
	}{
		{"current", SessionSelection{Scope: SessionCurrent}, true},
		{"others", SessionSelection{Scope: SessionOthers}, true},
		{"all", SessionSelection{Scope: SessionAll}, true},
		{"selected maximum", SessionSelection{Scope: SessionSelected, ID: strings.Repeat("a", 128)}, true},
		{"zero", SessionSelection{}, false}, {"unknown", SessionSelection{Scope: 255}, false},
		{"empty selected", SessionSelection{Scope: SessionSelected}, false},
		{"extra id", SessionSelection{Scope: SessionAll, ID: "other"}, false},
		{"oversized", SessionSelection{Scope: SessionSelected, ID: strings.Repeat("a", 129)}, false},
		{"control", SessionSelection{Scope: SessionSelected, ID: "a\nb"}, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if (test.selection.Check() == nil) != test.valid {
				t.Fatal("selection boundary")
			}
		})
	}

	for _, id := range []string{"record", strings.Repeat("z", 96)} {
		cursor, err := SessionCursor(id)

		decoded, parseErr := ParseSessionCursor(cursor)
		if err != nil || parseErr != nil || decoded != id || len(cursor) > 128 {
			t.Fatal("cursor round trip")
		}
	}

	for _, cursor := range []string{"=", "YQ==", "YR", strings.Repeat("a", 129), "Cg"} {
		_, err := ParseSessionCursor(cursor)
		if !errors.Is(err, ErrSessionCursor) {
			t.Fatal("malformed cursor accepted")
		}
	}
}

// Actual service verification refreshes proof, while failed work preserves all old metadata.
func TestReauthentication(t *testing.T) {
	q := newMockQueries()
	svc := newServiceForTest(t, q, config.New(), log.NewTestLogger("error"))

	_, err := svc.Signup(t.Context(), "control@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	issued, err := testSignin(svc, t.Context(), "control@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	digest, _ := ParseSessionToken(issued.Token)
	old := q.sessions[digest]
	old.CreatedAt = old.CreatedAt.Add(-10 * time.Minute)
	old.AuthenticatedAt = old.AuthenticatedAt.Add(-10 * time.Minute)
	old.Proof.VerifiedAt = old.AuthenticatedAt
	before := *old
	required := testRequirement()

	required.MaxAge = time.Second

	_, err = svc.ListSessions(t.Context(), issued.Token, required, "")
	if !errors.Is(err, ErrSessionProofExpired) {
		t.Fatal("stale management proof accepted")
	}

	outcome, err := svc.Reauthenticate(t.Context(), issued.Token, "wrong password", required)
	if outcome != nil || !errors.Is(err, ErrInvalidPassword) || !reflect.DeepEqual(before, *q.sessions[digest]) {
		t.Fatal("bad proof changed session")
	}

	entropy := svc.sessionToken
	failure := errors.New("entropy unavailable")
	svc.sessionToken = func() (string, SessionDigest, error) { return "", SessionDigest{}, failure }

	outcome, err = svc.Reauthenticate(t.Context(), issued.Token, "a distinct safe password", required)
	if outcome != nil || !errors.Is(err, failure) || !reflect.DeepEqual(before, *q.sessions[digest]) {
		t.Fatal("entropy failure changed session")
	}

	svc.sessionToken = entropy
	outcome, err = svc.Reauthenticate(t.Context(), issued.Token, "a distinct safe password", required)

	fresh, ok := outcome.CompletedSession()
	if err != nil || !ok || fresh.Token == issued.Token || fresh.Generation != before.Generation+1 || fresh.ID != before.ID || !fresh.CreatedAt.Equal(before.CreatedAt) || !fresh.Proof.VerifiedAt.After(before.Proof.VerifiedAt) {
		t.Fatalf("rotation: %v", err)
	}

	_, err = svc.ValidateSession(t.Context(), issued.Token, testRequirement(), NoActivity)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatal("old bearer still live")
	}

	_, err = svc.ListSessions(t.Context(), fresh.Token, required, "")
	if err != nil {
		t.Fatal(err)
	}

	strong := required
	strong.Proof = RequirePhishingResistantMFA

	outcome, err = svc.Reauthenticate(t.Context(), fresh.Token, "a distinct safe password", strong)
	if outcome != nil || !errors.Is(err, ErrSessionProof) {
		t.Fatal("password met stronger policy")
	}

	newDigest, _ := ParseSessionToken(fresh.Token)
	q.sessions[newDigest].Generation = math.MaxInt64

	outcome, err = svc.Reauthenticate(t.Context(), fresh.Token, "a distinct safe password", required)
	if outcome != nil || !errors.Is(err, ErrSessionGeneration) {
		t.Fatal("generation overflow accepted")
	}
}

type controlDeadlineQueries struct {
	*mockQueries
	deadline time.Time
}

func (q *controlDeadlineQueries) RevokeSessions(ctx context.Context, digest SessionDigest, requirement AccessRequirement, selection SessionSelection) (int64, error) {
	q.deadline, _ = ctx.Deadline()
	<-ctx.Done()

	return 1, nil
}

// A late successful adapter result cannot report revocation after either deadline.
func TestControlDeadline(t *testing.T) {
	for _, test := range []struct {
		name   string
		parent time.Duration
		want   time.Duration
	}{{"operation", time.Hour, time.Second}, {"parent", 100 * time.Millisecond, 100 * time.Millisecond}} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				q := &controlDeadlineQueries{mockQueries: newMockQueries()}
				cfg := config.New()
				cfg.Auth.SessionTimeout = "1s"
				svc := newServiceForTest(t, q, cfg, log.NewTestLogger("error"))

				token, _, err := newSessionToken()
				if err != nil {
					t.Fatal(err)
				}

				now := time.Now()

				ctx, cancel := context.WithTimeout(t.Context(), test.parent)
				defer cancel()

				count, err := svc.RevokeSessions(ctx, token, testRequirement(), SessionSelection{Scope: SessionAll})
				if count != 0 || !errors.Is(err, context.DeadlineExceeded) || q.deadline.Sub(now) != test.want {
					t.Fatal("late revocation result escaped")
				}
			})
		})
	}
}

// The service caps management freshness independently of weaker operation policy.
func TestManagementPolicy(t *testing.T) {
	svc := newServiceForTest(t, newMockQueries(), config.New(), log.NewTestLogger("error"))
	for _, test := range []struct {
		name      string
		age, want time.Duration
	}{{"default", 0, 5 * time.Minute}, {"looser", time.Hour, 5 * time.Minute}, {"tighter", time.Second, time.Second}} {
		t.Run(test.name, func(t *testing.T) {
			supplied := testRequirement()
			supplied.MaxAge = test.age

			actual, err := svc.managementRequirement(supplied)
			if err != nil || actual.MaxAge != test.want || supplied.MaxAge != test.age {
				t.Fatal("management policy weakened or mutated")
			}
		})
	}
}
