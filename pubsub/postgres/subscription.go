// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"hatmax.adrianpk.com/pubsub"
)

var (
	ErrSubscriptionConflict = errors.New("pubsub: conflicting durable subscription")
	ErrDeliveryClaim        = errors.New("pubsub: delivery is not claimable")
	ErrStaleDelivery        = errors.New("pubsub: stale delivery claim")
)

type subscriptionPolicy struct {
	handler, lease, delay int64
	attempts              int
	history               string
}

func (p subscriptionPolicy) execution() (pubsub.DeliveryPolicy, error) {
	if p.handler < 1000 || p.handler > 600000 || p.lease < p.handler+5000 || p.lease > 900000 || p.delay < 1000 || p.delay > 3600000 || p.attempts < 1 || p.attempts > 32 {
		return pubsub.DeliveryPolicy{}, pubsub.ErrDurableOptions
	}

	return pubsub.DeliveryPolicy{HandlerTimeout: time.Duration(p.handler) * time.Millisecond, LeaseDuration: time.Duration(p.lease) * time.Millisecond,
		MaxAttempts: p.attempts, RetryDelay: time.Duration(p.delay) * time.Millisecond}, nil
}

// ConfigureDurableSubscription explicitly upgrades an existing named subscriber
// or changes its policy for future deliveries. In-flight snapshots and successful
// acknowledgements remain unchanged. Existing durable history admission is fixed.
func (b *Broker) ConfigureDurableSubscription(ctx context.Context, subscriberID, topic string, opts pubsub.DurableOptions) error {
	return b.registerDurable(ctx, subscriberID, topic, opts, true)
}

func (b *Broker) registerDurable(ctx context.Context, id, topic string, opts pubsub.DurableOptions, configure bool) error {
	err := opts.Validate()
	if err != nil {
		return err
	}

	if !validIdentity(id) || !validIdentity(topic) {
		return pubsub.ErrDurableOptions
	}

	if b.db == nil {
		return ErrTopicNotAdmitted
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var capacity int
	// Serialize registration limits, independently of message publication slots.
	err = tx.QueryRowContext(ctx, `SELECT capacity FROM pubsub_bounded_config WHERE singleton FOR UPDATE`).Scan(&capacity)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTopicNotAdmitted
	}

	if err != nil {
		return err
	}

	var admitted bool

	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pubsub_topics WHERE topic=$1)`, topic).Scan(&admitted)
	if err != nil {
		return err
	}

	if !admitted {
		return ErrTopicNotAdmitted
	}

	var (
		original string
		bounded  bool
		retired  *time.Time
		stored   subscriptionPolicy
	)

	err = tx.QueryRowContext(ctx, `SELECT topic,bounded,COALESCE(handler_timeout_ms,0),COALESCE(lease_duration_ms,0),COALESCE(attempt_limit,0),COALESCE(retry_delay_ms,0),COALESCE(history_mode,''),retired_at
 FROM pubsub_subscriptions WHERE id=$1 FOR UPDATE`, id).Scan(&original, &bounded, &stored.handler, &stored.lease, &stored.attempts, &stored.delay, &stored.history, &retired)

	fresh := errors.Is(err, sql.ErrNoRows)
	if err != nil && !fresh {
		return err
	}

	if fresh && configure {
		return ErrSubscriptionConflict
	}

	if !fresh {
		if retired != nil {
			return ErrSubscriberRetired
		}

		if original != topic || (!bounded && !configure) {
			return ErrSubscriptionConflict
		}

		if bounded {
			policy, err := stored.execution()
			if err != nil {
				return err
			}

			if stored.history != string(opts.History) || (!configure && policy != opts.Policy) {
				return ErrSubscriptionConflict
			}

			if !configure {
				return tx.Commit()
			}
		}
	}

	if !bounded {
		var count int

		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM pubsub_subscriptions WHERE bounded`).Scan(&count)
		if err != nil {
			return err
		}

		if count >= 64 {
			return ErrAdmissionLimit
		}
	}

	policy := opts.Policy
	if fresh {
		_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_subscriptions(id,topic,acknowledgements_initialized,bounded,handler_timeout_ms,lease_duration_ms,attempt_limit,retry_delay_ms,history_mode)
 VALUES($1,$2,true,true,$3,$4,$5,$6,$7)`, id, topic, policy.HandlerTimeout.Milliseconds(), policy.LeaseDuration.Milliseconds(), policy.MaxAttempts, policy.RetryDelay.Milliseconds(), opts.History)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE pubsub_subscriptions SET bounded=true,handler_timeout_ms=$2,lease_duration_ms=$3,attempt_limit=$4,retry_delay_ms=$5,history_mode=$6,updated_at=clock_timestamp() WHERE id=$1`,
			id, policy.HandlerTimeout.Milliseconds(), policy.LeaseDuration.Milliseconds(), policy.MaxAttempts, policy.RetryDelay.Milliseconds(), opts.History)
	}

	if err != nil {
		return err
	}

	if (fresh || !bounded) && opts.History == pubsub.CurrentPublications {
		// This one statement sees only committed history. Lower IDs that commit later
		// remain eligible; no scalar high-water mark is an exclusion boundary.
		_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_acknowledgements(subscriber_id,message_id)
 SELECT $1,id FROM pubsub_messages WHERE topic=$2 AND bounded ON CONFLICT DO NOTHING`, id, topic)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func lockSubscriber(ctx context.Context, tx *sql.Tx, id string) (string, pubsub.DeliveryPolicy, error) {
	var (
		topic  string
		stored subscriptionPolicy
	)

	err := tx.QueryRowContext(ctx, `SELECT topic,handler_timeout_ms,lease_duration_ms,attempt_limit,retry_delay_ms
 FROM pubsub_subscriptions WHERE id=$1 AND bounded AND retired_at IS NULL FOR UPDATE`, id).Scan(&topic, &stored.handler, &stored.lease, &stored.attempts, &stored.delay)
	if errors.Is(err, sql.ErrNoRows) {
		return "", pubsub.DeliveryPolicy{}, ErrDeliveryClaim
	}

	if err != nil {
		return "", pubsub.DeliveryPolicy{}, err
	}

	policy, err := stored.execution()

	return topic, policy, err
}
