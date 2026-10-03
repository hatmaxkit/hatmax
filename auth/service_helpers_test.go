// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
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
