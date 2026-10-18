// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package pubsub

import (
	"errors"
	"time"
)

var ErrDurableOptions = errors.New("pubsub: invalid durable subscription options")

// DeliveryPolicy is frozen on each delivery's first claim. All durations must
// be whole milliseconds; handler cancellation is cooperative.
type DeliveryPolicy struct {
	HandlerTimeout time.Duration
	LeaseDuration  time.Duration
	MaxAttempts    int
	RetryDelay     time.Duration
}

func DefaultDeliveryPolicy() DeliveryPolicy {
	return DeliveryPolicy{HandlerTimeout: 10 * time.Second, LeaseDuration: 30 * time.Second, MaxAttempts: 8, RetryDelay: 30 * time.Second}
}

func (p DeliveryPolicy) Validate() error {
	if p.HandlerTimeout < time.Second || p.HandlerTimeout > 10*time.Minute || p.LeaseDuration < p.HandlerTimeout+5*time.Second ||
		p.LeaseDuration > 15*time.Minute || p.MaxAttempts < 1 || p.MaxAttempts > 32 || p.RetryDelay < time.Second || p.RetryDelay > time.Hour ||
		p.HandlerTimeout%time.Millisecond != 0 || p.LeaseDuration%time.Millisecond != 0 || p.RetryDelay%time.Millisecond != 0 {
		return ErrDurableOptions
	}

	return nil
}

type HistoryMode string

const (
	CurrentPublications HistoryMode = "current"
	RetainedHistory     HistoryMode = "retained"
)

// DurableOptions explicitly chooses history and a fixed policy for a named
// subscriber. Existing successful acknowledgements are never implicitly replayed.
type DurableOptions struct {
	Policy  DeliveryPolicy
	History HistoryMode
}

func (o DurableOptions) Validate() error {
	err := o.Policy.Validate()
	if err != nil {
		return err
	}

	if o.History != CurrentPublications && o.History != RetainedHistory {
		return ErrDurableOptions
	}

	return nil
}
