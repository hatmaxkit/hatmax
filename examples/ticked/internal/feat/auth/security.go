// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"encoding/json"
	"errors"

	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/log"
)

// SecurityLogger writes only the checked, closed security event shape.
// The application owns logger retention and delivery; this is not an audit outbox.
// Its logger must honor the cooperative callback deadline and be concurrency-safe.
type SecurityLogger struct{ Logger log.Logger }

func (s SecurityLogger) Observe(ctx context.Context, event auth.SecurityEvent) error {
	if s.Logger == nil {
		return errors.New("security logger is required")
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	err = event.Check()
	if err != nil {
		return auth.ErrSecurityObservationRejected
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return auth.ErrSecurityObservationRejected
	}

	s.Logger.Infof("Authentication security event: %s", payload)

	return ctx.Err()
}
