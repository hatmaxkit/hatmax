// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"sort"
	"time"
)

// These sequential fakes support service tests; real SQL tests own atomicity evidence.
func (m *mockQueries) RotateSession(ctx context.Context, state CredentialState, digest SessionDigest, generation int64, record SessionRecord, requirement AccessRequirement) (*Session, error) {
	currentRequirement := requirement
	currentRequirement.MaxAge = 0

	current, err := m.ValidateSession(ctx, digest, currentRequirement, NoActivity, time.Second)
	if err != nil {
		return nil, err
	}

	if current.User.AuthVersion != state.Version {
		return nil, ErrCredentialChanged
	}

	if current.Session.Generation != generation {
		return nil, ErrSessionGeneration
	}

	err = requirement.Evaluate(record.Session, time.Now())
	if err != nil {
		return nil, err
	}

	delete(m.sessions, digest)
	m.sessions[record.Digest] = &record
	snapshot := record.Session

	return &snapshot, nil
}
func (m *mockQueries) ListSessions(ctx context.Context, digest SessionDigest, requirement AccessRequirement, limit int, cursor string) (*SessionPage, error) {
	actor, err := m.ValidateSession(ctx, digest, requirement, NoActivity, time.Second)
	if err != nil {
		return nil, err
	}

	after, err := ParseSessionCursor(cursor)
	if err != nil {
		return nil, err
	}

	page := &SessionPage{CurrentID: actor.Session.ID}
	for _, record := range m.sessions {
		if record.UserID == actor.User.ID && record.ID > after {
			page.Sessions = append(page.Sessions, record.Session)
		}
	}

	sort.Slice(page.Sessions, func(i, j int) bool { return page.Sessions[i].ID < page.Sessions[j].ID })

	if len(page.Sessions) > limit {
		page.Sessions = page.Sessions[:limit]
		page.NextCursor, err = SessionCursor(page.Sessions[limit-1].ID)
	}

	return page, err
}
func (m *mockQueries) RevokeSessions(ctx context.Context, digest SessionDigest, requirement AccessRequirement, selection SessionSelection) (int64, error) {
	actor, err := m.ValidateSession(ctx, digest, requirement, NoActivity, time.Second)
	if err != nil {
		return 0, err
	}

	var count int64

	for key, record := range m.sessions {
		match := selection.Scope == SessionAll || (selection.Scope == SessionCurrent && record.ID == actor.Session.ID) || (selection.Scope == SessionOthers && record.ID != actor.Session.ID) || (selection.Scope == SessionSelected && record.ID == selection.ID)
		if record.UserID == actor.User.ID && match {
			delete(m.sessions, key)

			count++
		}
	}

	if selection.Scope == SessionSelected && count == 0 {
		return 0, ErrSessionNotFound
	}

	return count, nil
}
