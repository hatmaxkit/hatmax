// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"fmt"
	"testing"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

// Unit tests use an explicit accepting fake; it does not represent breach coverage.
func newServiceForTest(t *testing.T, q Queries, cfg *config.Config, logger log.Logger) *Service {
	t.Helper()

	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1

	svc, err := NewService(q, cfg, passwordCheckerFunc(func(context.Context, string) (bool, error) { return false, nil }), logger)
	if err != nil {
		t.Fatal(err)
	}

	return svc
}

func testRequirement() AccessRequirement {
	return AccessRequirement{Proof: RequirePassword, Revision: "password-v1"}
}

func testSignin(svc *Service, ctx context.Context, email, password string) (*IssuedSession, error) {
	result, err := svc.Signin(ctx, email, password, testRequirement())
	if err != nil {
		return nil, err
	}

	issued, ok := result.CompletedSession()
	if !ok {
		return nil, fmt.Errorf("unexpected test authentication outcome")
	}

	return issued, nil
}
