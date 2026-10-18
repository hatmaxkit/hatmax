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

	"github.com/lib/pq"
)

var (
	ErrWorkQuery         = errors.New("pubsub: invalid work query or operation bounds")
	ErrReplayConflict    = errors.New("pubsub: delivery revision cannot be replayed")
	ErrSubscriberRetired = errors.New("pubsub: subscriber is retired")
	ErrPendingDelivery   = errors.New("pubsub: subscriber has unfinished deliveries")
)

// DeliveryCursor is a stable keyset position from a previous inspection row.
type DeliveryCursor struct {
	SubscriberID string
	MessageID    int64
}

// DeliveryQuery returns 50 rows by default and at most 100. State is empty,
// pending, running, expired, retry_wait, succeeded or failed.
type DeliveryQuery struct {
	SubscriberID string
	Topic        string
	State        string
	Limit        int
	After        DeliveryCursor
}

// DeliveryInfo contains bounded diagnostic data without payload, metadata or
// ownership tokens. Pending work has zero attempts and no invented policy budget.
type DeliveryInfo struct {
	SubscriberID string
	MessageID    int64
	EnvelopeID   string
	Topic        string
	State        string
	Attempt      int
	AttemptLimit int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	FinishedAt   *time.Time
	EligibleAt   *time.Time
	FailureCode  string
	ReplayedAt   *time.Time
	ReplayReason string
}

func (b *Broker) ListDeliveries(ctx context.Context, query DeliveryQuery) ([]DeliveryInfo, error) {
	if query.Limit == 0 {
		query.Limit = 50
	}

	if query.Limit < 1 || query.Limit > 100 || query.After.MessageID < 0 || (query.SubscriberID != "" && !validIdentity(query.SubscriberID)) || (query.Topic != "" && !validIdentity(query.Topic)) || (query.After.SubscriberID != "" && !validIdentity(query.After.SubscriberID)) || (query.After.SubscriberID == "") != (query.After.MessageID == 0) {
		return nil, ErrWorkQuery
	}

	switch query.State {
	case "", "pending", "running", "expired", "retry_wait", "succeeded", "failed":
	default:
		return nil, ErrWorkQuery
	}

	rows, err := b.db.QueryContext(ctx, `WITH work AS (
 SELECT s.id AS subscriber_id,m.id AS message_id,m.message_id AS envelope_id,m.topic,
 CASE WHEN d.message_id IS NULL THEN 'pending' WHEN d.state='running' AND d.lease_expires_at<=clock_timestamp() THEN 'expired' ELSE d.state END AS state,
 COALESCE(d.attempt,0) AS attempt,COALESCE(d.attempt_limit,0) AS attempt_limit,COALESCE(d.created_at,m.created_at) AS created_at,
 COALESCE(d.updated_at,m.created_at) AS updated_at,d.finished_at,COALESCE(d.retry_at,d.lease_expires_at) AS eligible_at,
 COALESCE(d.failure_code,'') AS failure_code,d.replayed_at,COALESCE(d.replay_reason,'') AS replay_reason
 FROM pubsub_subscriptions s JOIN pubsub_messages m ON m.topic=s.topic AND m.bounded
 LEFT JOIN pubsub_deliveries d ON d.subscriber_id=s.id AND d.message_id=m.id
 WHERE s.bounded AND (d.message_id IS NOT NULL OR (s.retired_at IS NULL AND NOT EXISTS(
 SELECT 1 FROM pubsub_acknowledgements a WHERE a.subscriber_id=s.id AND a.message_id=m.id))))
 SELECT subscriber_id,message_id,envelope_id,topic,state,attempt,attempt_limit,created_at,updated_at,finished_at,eligible_at,failure_code,replayed_at,replay_reason
 FROM work WHERE ($1='' OR subscriber_id=$1) AND ($2='' OR topic=$2) AND ($3='' OR state=$3)
 AND ($4='' OR (subscriber_id,message_id)>($4,$5)) ORDER BY subscriber_id,message_id LIMIT $6`,
		query.SubscriberID, query.Topic, query.State, query.After.SubscriberID, query.After.MessageID, query.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []DeliveryInfo

	for rows.Next() {
		var row DeliveryInfo

		err = rows.Scan(&row.SubscriberID, &row.MessageID, &row.EnvelopeID, &row.Topic, &row.State, &row.Attempt, &row.AttemptLimit, &row.CreatedAt, &row.UpdatedAt, &row.FinishedAt, &row.EligibleAt, &row.FailureCode, &row.ReplayedAt, &row.ReplayReason)
		if err != nil {
			return nil, err
		}

		deliveries = append(deliveries, row)
	}

	return deliveries, rows.Err()
}

type ReplayReason string

const (
	ReplayOperator   ReplayReason = "operator_retry"
	ReplayDependency ReplayReason = "dependency_restored"
	ReplayDataRepair ReplayReason = "data_repaired"
)

// ReplayRequest authorizes at most eight additional attempts within the lifetime
// maximum of 32 and matches the inspected terminal attempt and update timestamp.
type ReplayRequest struct {
	SubscriberID       string
	MessageID          int64
	Attempt            int
	UpdatedAt          time.Time
	AdditionalAttempts int
	Reason             ReplayReason
}

func (b *Broker) ReplayDelivery(ctx context.Context, request ReplayRequest) error {
	if !validIdentity(request.SubscriberID) || request.MessageID < 1 || request.Attempt < 1 || request.Attempt > 31 || request.UpdatedAt.IsZero() ||
		request.AdditionalAttempts < 1 || request.AdditionalAttempts > 8 || request.Attempt+request.AdditionalAttempts > 32 {
		return ErrWorkQuery
	}

	switch request.Reason {
	case ReplayOperator, ReplayDependency, ReplayDataRepair:
	default:
		return ErrWorkQuery
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _, err = lockSubscriber(ctx, tx, request.SubscriberID)
	if errors.Is(err, ErrDeliveryClaim) {
		return ErrReplayConflict
	}

	if err != nil {
		return err
	}

	var id int64
	// Match claim lock order: subscriber, message, delivery. This also fences prune.
	err = tx.QueryRowContext(ctx, `SELECT id FROM pubsub_messages WHERE id=$1 AND bounded FOR KEY SHARE`, request.MessageID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrReplayConflict
	}

	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `UPDATE pubsub_deliveries d SET state='retry_wait',attempt_limit=attempt+$5,
 retry_at=clock_timestamp(),finished_at=NULL,lease_token=NULL,lease_expires_at=NULL,
 replayed_at=clock_timestamp(),replay_reason=$6,updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond')
 WHERE subscriber_id=$1 AND message_id=$2 AND state='failed' AND attempt=$3 AND updated_at=$4
 AND NOT EXISTS(SELECT 1 FROM pubsub_acknowledgements a WHERE a.subscriber_id=d.subscriber_id AND a.message_id=d.message_id)`,
		request.SubscriberID, request.MessageID, request.Attempt, request.UpdatedAt, request.AdditionalAttempts, request.Reason)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count != 1 {
		return ErrReplayConflict
	}

	return tx.Commit()
}

// RetireOptions requires an explicit choice to discard unfinished deliveries.
// Quiesce handlers before discarding; retirement fences but cannot terminate them.
type RetireOptions struct{ DiscardPending bool }

func (b *Broker) RetireSubscriber(ctx context.Context, id string, options RetireOptions) error {
	if !validIdentity(id) {
		return ErrWorkQuery
	}

	tx, err := b.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var capacity int

	err = tx.QueryRowContext(ctx, `SELECT capacity FROM pubsub_bounded_config WHERE singleton FOR UPDATE`).Scan(&capacity)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSubscriptionConflict
	}

	if err != nil {
		return err
	}
	// Serialize the entitlement boundary with admitted publications and registrations.
	_, err = tx.ExecContext(ctx, `LOCK TABLE pubsub_messages IN SHARE ROW EXCLUSIVE MODE`)
	if err != nil {
		return err
	}

	var (
		topic   string
		retired *time.Time
	)

	err = tx.QueryRowContext(ctx, `SELECT topic,retired_at FROM pubsub_subscriptions WHERE id=$1 AND bounded FOR UPDATE`, id).Scan(&topic, &retired)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSubscriptionConflict
	}

	if err != nil {
		return err
	}

	if retired != nil {
		return tx.Commit()
	}

	var unfinished bool

	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pubsub_messages m WHERE m.bounded AND m.topic=$2
 AND NOT EXISTS(SELECT 1 FROM pubsub_acknowledgements a WHERE a.subscriber_id=$1 AND a.message_id=m.id)
 AND NOT EXISTS(SELECT 1 FROM pubsub_deliveries d WHERE d.subscriber_id=$1 AND d.message_id=m.id AND d.state IN ('succeeded','failed')))`, id, topic).Scan(&unfinished)
	if err != nil {
		return err
	}

	if unfinished && !options.DiscardPending {
		return ErrPendingDelivery
	}

	_, err = tx.ExecContext(ctx, `UPDATE pubsub_deliveries SET state='failed',failure_code='subscriber_retired',lease_token=NULL,lease_expires_at=NULL,retry_at=NULL,
 finished_at=clock_timestamp(),updated_at=clock_timestamp() WHERE subscriber_id=$1 AND state IN ('running','retry_wait')`, id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE pubsub_subscriptions SET retired_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Include every still-entitled subscriber, including a legacy subscriber awaiting
// explicit upgrade. Acknowledged history needs no fabricated delivery row.
const pruneEligible = `m.bounded AND m.created_at<$1
 AND NOT EXISTS(SELECT 1 FROM pubsub_deliveries d WHERE d.message_id=m.id
 AND (d.state IN ('running','retry_wait') OR d.finished_at>=$1))
 AND NOT EXISTS(SELECT 1 FROM pubsub_subscriptions s WHERE s.topic=m.topic AND s.retired_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM pubsub_acknowledgements a WHERE a.subscriber_id=s.id AND a.message_id=m.id)
 AND NOT EXISTS(SELECT 1 FROM pubsub_deliveries d WHERE d.subscriber_id=s.id AND d.message_id=m.id AND d.state IN ('succeeded','failed')))`

// PruneMessages deletes at most 1000 eligible bounded messages and their tracking
// rows atomically, releasing capacity in the same transaction. It preserves all
// unfinished fan-out and terminal outcomes newer than the caller's cutoff.
func (b *Broker) PruneMessages(ctx context.Context, before time.Time, limit int) (int64, error) {
	if before.IsZero() || limit < 1 || limit > 1000 {
		return 0, ErrWorkQuery
	}

	tx, err := b.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var capacity int

	err = tx.QueryRowContext(ctx, `SELECT capacity FROM pubsub_bounded_config WHERE singleton FOR UPDATE`).Scan(&capacity)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrTopicNotAdmitted
	}

	if err != nil {
		return 0, err
	}

	rows, err := tx.QueryContext(ctx, `SELECT m.id FROM pubsub_messages m WHERE `+pruneEligible+` ORDER BY m.created_at,m.id LIMIT $2 FOR UPDATE OF m SKIP LOCKED`, before, limit)
	if err != nil {
		return 0, err
	}

	var ids []int64

	for rows.Next() {
		var id int64

		err = rows.Scan(&id)
		if err != nil {
			rows.Close()

			return 0, err
		}

		ids = append(ids, id)
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return 0, err
	}

	if len(ids) == 0 {
		return 0, tx.Commit()
	}
	// Recheck in a fresh snapshot after locking messages: a replay can commit
	// between the first eligibility snapshot and acquiring its unchanged message row.
	result, err := tx.ExecContext(ctx, `DELETE FROM pubsub_messages m WHERE m.id=ANY($2::bigint[]) AND `+pruneEligible, before, pq.Array(ids))
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	err = tx.Commit()

	return count, err
}
