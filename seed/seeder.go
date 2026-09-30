// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package seed

import "context"

// Seeder defines the interface for feature-specific seeders.
// Each feature implements its own seeder using the generic tools.
type Seeder interface {
	// Name returns a unique identifier for this seeder.
	// Used for tracking which seeds have been applied.
	Name() string

	// Seed executes the seeding logic for this feature.
	// Should be idempotent - called once and tracked in _seeds table.
	Seed(ctx context.Context) error
}
