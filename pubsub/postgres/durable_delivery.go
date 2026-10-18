// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"hatmax.adrianpk.com/pubsub"
)

type deliveryClaim struct {
	subscriber string
	message    int64
	attempt    int
	token      string
	expires    time.Time
	policy     pubsub.DeliveryPolicy
}

func (b *Broker) pollDurable(ctx context.Context, sub *subscription) error {
	rows, err := b.db.QueryContext(ctx, `SELECT m.id FROM pubsub_messages m
 LEFT JOIN pubsub_deliveries d ON d.message_id=m.id AND d.subscriber_id=$2
 WHERE m.topic=$1 AND m.bounded AND EXISTS(SELECT 1 FROM pubsub_subscriptions s WHERE s.id=$2 AND s.bounded AND s.retired_at IS NULL) AND NOT EXISTS(SELECT 1 FROM pubsub_acknowledgements a WHERE a.subscriber_id=$2 AND a.message_id=m.id)
 AND (d.message_id IS NULL OR (d.state='retry_wait' AND d.retry_at<=clock_timestamp()) OR (d.state='running' AND d.lease_expires_at<=clock_timestamp()))
 ORDER BY m.id LIMIT $3`, sub.topic, sub.id, min(b.cfg.BatchSize, 100))
	if err != nil {
		return err
	}

	var ids []int64

	for rows.Next() {
		var id int64

		err = rows.Scan(&id)
		if err != nil {
			rows.Close()

			return err
		}

		ids = append(ids, id)
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return err
	}

	for _, id := range ids {
		err = ctx.Err()
		if err != nil {
			return err
		}

		claim, env, code, err := b.claimDelivery(ctx, sub.id, id)
		if errors.Is(err, ErrDeliveryClaim) {
			continue
		}

		if err != nil {
			return err
		}

		if code == "" {
			handlerCtx, cancel := context.WithTimeout(ctx, claim.policy.HandlerTimeout)
			code = invokeDurable(handlerCtx, sub.handler, env)

			cancel()
		}

		err = b.completeDelivery(ctx, claim, code)
		if err != nil {
			return err
		}
	}

	return ctx.Err()
}

// Invoke on the polling goroutine. Deadlines do not spawn replacement handlers
// and no panic value or arbitrary error text enters retained diagnostics.
func invokeDurable(ctx context.Context, handler pubsub.Handler, env pubsub.Envelope) (code string) {
	completed := false

	defer func() {
		recover()

		if !completed {
			code = "handler_panic"
		}
	}()

	err := handler(ctx, env)
	completed = true

	switch {
	case ctx.Err() != nil:
		return "handler_deadline"
	case err != nil:
		return "handler_error"
	default:
		return ""
	}
}

func (b *Broker) claimDelivery(ctx context.Context, subscriber string, message int64) (deliveryClaim, pubsub.Envelope, string, error) {
	var empty deliveryClaim

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return empty, pubsub.Envelope{}, "", err
	}
	defer tx.Rollback()

	topic, policy, err := lockSubscriber(ctx, tx, subscriber)
	if err != nil {
		return empty, pubsub.Envelope{}, "", err
	}

	var (
		env               pubsub.Envelope
		payload, metadata []byte
	)

	err = tx.QueryRowContext(ctx, `SELECT message_id,topic,payload,metadata,COALESCE(occurred_at,created_at) FROM pubsub_messages m
 WHERE id=$1 AND topic=$2 AND bounded AND NOT EXISTS(SELECT 1 FROM pubsub_acknowledgements a WHERE a.subscriber_id=$3 AND a.message_id=m.id)
 FOR KEY SHARE OF m`, message, topic, subscriber).Scan(&env.ID, &env.Topic, &payload, &metadata, &env.Timestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return empty, pubsub.Envelope{}, "", ErrDeliveryClaim
	}

	if err != nil {
		return empty, pubsub.Envelope{}, "", err
	}

	claim := deliveryClaim{subscriber: subscriber, message: message, attempt: 1, policy: policy}

	var (
		state          string
		attempt        int
		expires, retry sql.NullTime
		stored         subscriptionPolicy
	)

	err = tx.QueryRowContext(ctx, `SELECT state,attempt,lease_expires_at,retry_at,handler_timeout_ms,lease_duration_ms,attempt_limit,retry_delay_ms
 FROM pubsub_deliveries WHERE subscriber_id=$1 AND message_id=$2 FOR UPDATE`, subscriber, message).Scan(&state, &attempt, &expires, &retry, &stored.handler, &stored.lease, &stored.attempts, &stored.delay)

	fresh := errors.Is(err, sql.ErrNoRows)
	if err != nil && !fresh {
		return empty, pubsub.Envelope{}, "", err
	}

	var now time.Time

	err = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now)
	if err != nil {
		return empty, pubsub.Envelope{}, "", err
	}

	if !fresh {
		if !((state == "running" && expires.Valid && !expires.Time.After(now)) || (state == "retry_wait" && retry.Valid && !retry.Time.After(now))) {
			return empty, pubsub.Envelope{}, "", ErrDeliveryClaim
		}

		policy, err = stored.execution()
		if err != nil {
			return empty, pubsub.Envelope{}, "", err
		}

		if attempt >= policy.MaxAttempts {
			_, err = tx.ExecContext(ctx, `UPDATE pubsub_deliveries SET state='failed',finished_at=$3,updated_at=$3,lease_token=NULL,lease_expires_at=NULL,retry_at=NULL,
 failure_code='lease_expired' WHERE subscriber_id=$1 AND message_id=$2`, subscriber, message, now)
			if err != nil {
				return empty, pubsub.Envelope{}, "", err
			}

			err = tx.Commit()
			if err != nil {
				return empty, pubsub.Envelope{}, "", err
			}

			return empty, pubsub.Envelope{}, "", ErrDeliveryClaim
		}

		claim.attempt, claim.policy = attempt+1, policy
	}

	claim.token, claim.expires = uuid.NewString(), now.Add(policy.LeaseDuration)
	if fresh {
		_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_deliveries(subscriber_id,message_id,state,attempt,attempt_limit,handler_timeout_ms,lease_duration_ms,retry_delay_ms,lease_token,lease_expires_at,created_at,updated_at)
 VALUES($1,$2,'running',1,$3,$4,$5,$6,$7,$8,$9,$9)`, subscriber, message, policy.MaxAttempts, policy.HandlerTimeout.Milliseconds(), policy.LeaseDuration.Milliseconds(), policy.RetryDelay.Milliseconds(), claim.token, claim.expires, now)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE pubsub_deliveries SET state='running',attempt=$3,lease_token=$4,lease_expires_at=$5,retry_at=NULL,finished_at=NULL,updated_at=$6,
 failure_code=CASE WHEN state='running' THEN 'lease_expired' ELSE failure_code END WHERE subscriber_id=$1 AND message_id=$2`, subscriber, message, claim.attempt, claim.token, claim.expires, now)
	}

	if err != nil {
		return empty, pubsub.Envelope{}, "", err
	}

	err = tx.Commit()
	if err != nil {
		return empty, pubsub.Envelope{}, "", err
	}

	code := decodeDurable(payload, metadata, &env)

	return claim, env, code, nil
}

func decodeDurable(payload, metadata []byte, env *pubsub.Envelope) string {
	data, err := canonicalJSON(payload)
	if err != nil || len(payload) > 65536 {
		return "invalid_payload"
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	err = decoder.Decode(&env.Payload)
	if err != nil {
		return "invalid_payload"
	}

	err = json.Unmarshal(metadata, &env.Metadata)
	if err != nil || env.Metadata == nil || len(metadata) > 8192 {
		return "invalid_metadata"
	}

	return ""
}

func validFailure(code string) bool {
	switch code {
	case "", "handler_error", "handler_panic", "handler_deadline", "invalid_payload", "invalid_metadata":
		return true
	default:
		return false
	}
}

func (b *Broker) completeDelivery(ctx context.Context, claim deliveryClaim, code string) error {
	if !validFailure(code) {
		return ErrEnvelope
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _, err = lockSubscriber(ctx, tx, claim.subscriber)
	if errors.Is(err, ErrDeliveryClaim) {
		return ErrStaleDelivery
	}

	if err != nil {
		return err
	}

	var (
		state, token          string
		expires               sql.NullTime
		attempt, limit, delay int
	)

	err = tx.QueryRowContext(ctx, `SELECT state,COALESCE(lease_token,''),lease_expires_at,attempt,attempt_limit,retry_delay_ms FROM pubsub_deliveries
 WHERE subscriber_id=$1 AND message_id=$2 FOR UPDATE`, claim.subscriber, claim.message).Scan(&state, &token, &expires, &attempt, &limit, &delay)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrStaleDelivery
	}

	if err != nil {
		return err
	}

	var now time.Time

	err = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now)
	if err != nil {
		return err
	}

	if state != "running" || claim.token == "" || token != claim.token || attempt != claim.attempt || !expires.Valid || !expires.Time.After(now) {
		return ErrStaleDelivery
	}

	if code == "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_acknowledgements(subscriber_id,message_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, claim.subscriber, claim.message)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `UPDATE pubsub_subscriptions SET last_message_id=GREATEST(last_message_id,$2),updated_at=$3 WHERE id=$1`, claim.subscriber, claim.message, now)
		if err != nil {
			return err
		}
	}

	if code != "" && attempt < limit {
		_, err = tx.ExecContext(ctx, `UPDATE pubsub_deliveries SET state='retry_wait',retry_at=$3,lease_token=NULL,lease_expires_at=NULL,finished_at=NULL,failure_code=$4,updated_at=$5
 WHERE subscriber_id=$1 AND message_id=$2`, claim.subscriber, claim.message, now.Add(time.Duration(delay)*time.Millisecond), code, now)
	} else {
		terminal := "succeeded"
		if code != "" {
			terminal = "failed"
		}

		_, err = tx.ExecContext(ctx, `UPDATE pubsub_deliveries SET state=$3,retry_at=NULL,lease_token=NULL,lease_expires_at=NULL,finished_at=$4,failure_code=NULLIF($5,''),updated_at=$4
 WHERE subscriber_id=$1 AND message_id=$2`, claim.subscriber, claim.message, terminal, now, code)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}
