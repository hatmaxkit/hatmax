// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/pubsub"
	"hatmax.adrianpk.com/testhelper"
)

// Real retained state protects revision-bound replay, finite lifetime budgets,
// explicit retirement, unfinished fan-out and transactional capacity reclamation.
func TestWorkOperations(t *testing.T) {
	t.Run("bounded inspection", deliveryInspection)
	t.Run("read only inspection", deliveryReadOnly)
	t.Run("replay lifetime", deliveryReplay)
	t.Run("fanout retention", deliveryRetention)
	t.Run("capacity reclamation", deliveryCapacity)
	t.Run("subscriber retirement", deliveryRetirement)
	t.Run("replay prune race", deliveryReplayPrune)
	t.Run("legacy upgrade retention", deliveryLegacyRetention)
}

// Operators can use an already-open read-only database without running Start's
// schema migration or admitting topics and subscriptions.
func deliveryReadOnly(t *testing.T) {
	b, db := boundedFixture(t)
	durableSub(t, b, "inspector", durableOptions(), nil)
	id := durableMessage(t, b, "inspection")

	db.SetMaxOpenConns(1)

	_, err := db.Exec(`SET default_transaction_read_only=on`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _, _ = db.Exec(`SET default_transaction_read_only=off`) }()

	inspector := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())

	rows, err := inspector.ListDeliveries(context.Background(), DeliveryQuery{SubscriberID: "inspector", Limit: 1})
	if err != nil || len(rows) != 1 || rows[0].MessageID != id {
		t.Fatalf("read-only inspection rows=%v error=%v", rows, err)
	}
}

func inspectedDelivery(t *testing.T, b *Broker, subscriber string, id int64) DeliveryInfo {
	t.Helper()

	rows, err := b.ListDeliveries(context.Background(), DeliveryQuery{SubscriberID: subscriber})
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range rows {
		if row.MessageID == id {
			return row
		}
	}

	t.Fatal("delivery missing from inspection")

	return DeliveryInfo{}
}

func finishDelivery(t *testing.T, b *Broker, subscriber string, id int64, code string) deliveryClaim {
	t.Helper()

	claim, _, decode, err := b.claimDelivery(context.Background(), subscriber, id)
	if err != nil || decode != "" {
		t.Fatalf("claim error=%v decode=%s", err, decode)
	}

	err = b.completeDelivery(context.Background(), claim, code)
	if err != nil {
		t.Fatal(err)
	}

	return claim
}

func dueDelivery(t *testing.T, db *sql.DB, subscriber string, id int64) {
	t.Helper()

	_, err := db.Exec(`UPDATE pubsub_deliveries SET retry_at=clock_timestamp()-interval '1 second' WHERE subscriber_id=$1 AND message_id=$2 AND state='retry_wait'`, subscriber, id)
	if err != nil {
		t.Fatal(err)
	}
}

func deliveryInspection(t *testing.T) {
	b, db := boundedFixture(t)
	durableMessage(t, b, "history")

	opts := durableOptions()
	opts.History = pubsub.CurrentPublications
	opts.Policy.MaxAttempts = 2
	durableSub(t, b, "worker", opts, nil)

	ids := map[string]int64{}
	for _, name := range []string{"pending", "running", "expired", "waiting", "failed", "success"} {
		ids[name] = durableMessage(t, b, name)
	}

	for _, name := range []string{"running", "expired"} {
		_, _, _, err := b.claimDelivery(context.Background(), "worker", ids[name])
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err := db.Exec(`UPDATE pubsub_deliveries SET lease_expires_at=clock_timestamp()-interval '1 second' WHERE message_id=$1`, ids["expired"])
	if err != nil {
		t.Fatal(err)
	}

	finishDelivery(t, b, "worker", ids["waiting"], "handler_error")
	finishDelivery(t, b, "worker", ids["failed"], "handler_error")
	dueDelivery(t, db, "worker", ids["failed"])
	finishDelivery(t, b, "worker", ids["failed"], "handler_error")
	finishDelivery(t, b, "worker", ids["success"], "")

	query := DeliveryQuery{Limit: 2}
	states := map[string]int{}
	seen := map[int64]bool{}

	for {
		page, err := b.ListDeliveries(context.Background(), query)
		if err != nil {
			t.Fatal(err)
		}

		if len(page) == 0 {
			break
		}

		for _, row := range page {
			if seen[row.MessageID] {
				t.Fatal("keyset repeated a delivery")
			}

			seen[row.MessageID] = true

			states[row.State]++
			if row.State == "pending" && (row.Attempt != 0 || row.AttemptLimit != 0) {
				t.Fatal("unclaimed work has invented attempt history")
			}

			if row.State == "expired" && row.EligibleAt == nil {
				t.Fatal("expired work omitted its eligible time")
			}
		}

		last := page[len(page)-1]
		query.After = DeliveryCursor{last.SubscriberID, last.MessageID}
	}

	if len(seen) != 6 || len(states) != 6 {
		t.Fatalf("inspection rows=%d states=%v", len(seen), states)
	}

	for _, bad := range []DeliveryQuery{{Limit: 101}, {Limit: -1}, {State: "unknown"}, {After: DeliveryCursor{SubscriberID: "partial"}}, {After: DeliveryCursor{MessageID: 1}}} {
		_, err = b.ListDeliveries(context.Background(), bad)
		if !errors.Is(err, ErrWorkQuery) {
			t.Fatalf("invalid query accepted: %v", err)
		}
	}

	filtered, err := b.ListDeliveries(context.Background(), DeliveryQuery{SubscriberID: "worker", Topic: "events", State: "failed"})
	if err != nil || len(filtered) != 1 || filtered[0].FailureCode != "handler_error" {
		t.Fatalf("filtered rows=%d error=%v", len(filtered), err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = b.ListDeliveries(canceled, DeliveryQuery{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("inspection ignored cancellation: %v", err)
	}
}

func deliveryReplay(t *testing.T) {
	b, db := boundedFixture(t)
	opts := durableOptions()
	opts.Policy.MaxAttempts = 2
	opts.Policy.RetryDelay = time.Second
	durableSub(t, b, "worker", opts, nil)
	id := durableMessage(t, b, "failed")
	finishDelivery(t, b, "worker", id, "handler_error")
	dueDelivery(t, db, "worker", id)
	old := finishDelivery(t, b, "worker", id, "handler_error")
	observed := inspectedDelivery(t, b, "worker", id)

	request := ReplayRequest{"worker", id, observed.Attempt, observed.UpdatedAt, 2, ReplayOperator}
	for _, grant := range []int{-1, 0, 9} {
		bad := request
		bad.AdditionalAttempts = grant

		err := b.ReplayDelivery(context.Background(), bad)
		if !errors.Is(err, ErrWorkQuery) {
			t.Fatalf("invalid replay grant accepted: %v", err)
		}
	}

	badReason := request
	badReason.Reason = "arbitrary text"

	err := b.ReplayDelivery(context.Background(), badReason)
	if !errors.Is(err, ErrWorkQuery) {
		t.Fatalf("unclassified replay reason accepted: %v", err)
	}

	var (
		wins  atomic.Int32
		group sync.WaitGroup
	)
	for range 12 {
		group.Go(func() {
			err := b.ReplayDelivery(context.Background(), request)
			if err == nil {
				wins.Add(1)
			} else if !errors.Is(err, ErrReplayConflict) {
				t.Error(err)
			}
		})
	}

	group.Wait()

	if wins.Load() != 1 {
		t.Fatalf("replay winners=%d", wins.Load())
	}

	row := inspectedDelivery(t, b, "worker", id)
	if row.Attempt != 2 || row.AttemptLimit != 4 || row.State != "retry_wait" || row.ReplayedAt == nil || row.ReplayReason != string(ReplayOperator) || !row.UpdatedAt.After(observed.UpdatedAt) {
		t.Fatalf("replay reset counters or lost revision metadata: attempt=%d limit=%d state=%s", row.Attempt, row.AttemptLimit, row.State)
	}

	err = b.completeDelivery(context.Background(), old, "")
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("old owner acknowledged replay: %v", err)
	}
	// Exhaust real attempts through successive replay grants up to the lifetime cap.
	target := 4
	for _, nextGrant := range []int{8, 8, 8, 4, 0} {
		for attempt := row.Attempt + 1; attempt <= target; attempt++ {
			dueDelivery(t, db, "worker", id)

			claim := finishDelivery(t, b, "worker", id, "handler_error")
			if claim.attempt != attempt || claim.policy.MaxAttempts != target || claim.policy.RetryDelay != opts.Policy.RetryDelay {
				t.Fatalf("replayed attempt=%d limit=%d", claim.attempt, claim.policy.MaxAttempts)
			}
		}

		row = inspectedDelivery(t, b, "worker", id)
		if row.State != "failed" || row.Attempt != target {
			t.Fatalf("grant did not terminate: state=%s attempt=%d", row.State, row.Attempt)
		}

		if nextGrant == 0 {
			break
		}

		err = b.ReplayDelivery(context.Background(), ReplayRequest{"worker", id, row.Attempt, row.UpdatedAt, nextGrant, ReplayDependency})
		if err != nil {
			t.Fatal(err)
		}

		row = inspectedDelivery(t, b, "worker", id)
		target += nextGrant
	}

	if row.Attempt != 32 {
		t.Fatalf("lifetime attempt=%d", row.Attempt)
	}

	err = b.ReplayDelivery(context.Background(), request)
	if !errors.Is(err, ErrReplayConflict) {
		t.Fatalf("stale revision reset budget: %v", err)
	}

	for _, grant := range []int{-1, 0, 1, 9} {
		err = b.ReplayDelivery(context.Background(), ReplayRequest{"worker", id, row.Attempt, row.UpdatedAt, grant, ReplayOperator})
		if !errors.Is(err, ErrWorkQuery) {
			t.Fatalf("lifetime or grant bound bypassed: %v", err)
		}
	}

	missing := ReplayRequest{"worker", id + 100000, 1, time.Now(), 1, ReplayOperator}

	err = b.ReplayDelivery(context.Background(), missing)
	if !errors.Is(err, ErrReplayConflict) {
		t.Fatalf("missing delivery replay accepted: %v", err)
	}

	history := opts
	history.History = pubsub.CurrentPublications
	durableSub(t, b, "history-skipped", history, nil)

	err = b.ReplayDelivery(context.Background(), ReplayRequest{"history-skipped", id, 1, time.Now(), 1, ReplayOperator})
	if !errors.Is(err, ErrReplayConflict) {
		t.Fatalf("acknowledged history replay accepted: %v", err)
	}
	// Acknowledged, unclaimed and active work cannot be replayed as a failure.
	err = nil

	for _, state := range []string{"pending", "running", "succeeded", "retry_wait"} {
		other := durableMessage(t, b, state)
		if state == "running" {
			_, _, _, err = b.claimDelivery(context.Background(), "worker", other)
		}

		if state == "succeeded" {
			finishDelivery(t, b, "worker", other, "")
		}

		if state == "retry_wait" {
			finishDelivery(t, b, "worker", other, "handler_error")
		}

		if err != nil {
			t.Fatal(err)
		}

		current := inspectedDelivery(t, b, "worker", other)

		err = b.ReplayDelivery(context.Background(), ReplayRequest{"worker", other, max(current.Attempt, 1), current.UpdatedAt, 1, ReplayDataRepair})
		if !errors.Is(err, ErrReplayConflict) {
			t.Fatalf("%s replay accepted: %v", state, err)
		}

		err = nil
	}
}

func deliveryRetention(t *testing.T) {
	b, db := boundedFixture(t)
	opts := durableOptions()
	opts.Policy.MaxAttempts = 2
	durableSub(t, b, "first", opts, nil)
	durableSub(t, b, "second", opts, nil)

	ids := map[string]int64{}
	for _, name := range []string{"pending", "running", "waiting", "failed", "succeeded"} {
		ids[name] = durableMessage(t, b, name)
		finishDelivery(t, b, "first", ids[name], "")
	}

	_, _, _, err := b.claimDelivery(context.Background(), "second", ids["running"])
	if err != nil {
		t.Fatal(err)
	}

	finishDelivery(t, b, "second", ids["waiting"], "handler_error")
	finishDelivery(t, b, "second", ids["failed"], "handler_error")
	dueDelivery(t, db, "second", ids["failed"])
	finishDelivery(t, b, "second", ids["failed"], "handler_error")
	finishDelivery(t, b, "second", ids["succeeded"], "")

	current := opts
	current.History = pubsub.CurrentPublications
	durableSub(t, b, "history-excluded", current, nil)

	_, err = db.Exec(`UPDATE pubsub_messages SET created_at=clock_timestamp()-interval '2 days'`)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := b.PruneMessages(ctx, time.Now().Add(-time.Hour), 1000)
	if err != nil || count != 0 {
		t.Fatalf("recent terminal history expired: %d %v", count, err)
	}

	count, err = b.PruneMessages(ctx, time.Now().Add(time.Hour), 1)
	if err != nil || count != 1 {
		t.Fatalf("bounded prune=%d error=%v", count, err)
	}

	count, err = b.PruneMessages(ctx, time.Now().Add(time.Hour), 1000)
	if err != nil || count != 1 {
		t.Fatalf("remaining terminal prune=%d error=%v", count, err)
	}

	var messages, slots, terminal int

	err = db.QueryRow(`SELECT (SELECT count(*) FROM pubsub_messages),(SELECT count(*) FROM pubsub_capacity_slots WHERE message_id IS NOT NULL),(SELECT count(*) FROM pubsub_deliveries WHERE message_id=$1 OR message_id=$2)`, ids["failed"], ids["succeeded"]).Scan(&messages, &slots, &terminal)
	if err != nil || messages != 3 || slots != 3 || terminal != 0 {
		t.Fatalf("pruning lost unfinished fanout or tracking: messages=%d slots=%d terminal=%d error=%v", messages, slots, terminal, err)
	}

	for _, limit := range []int{-1, 0, 1001} {
		_, err = b.PruneMessages(ctx, time.Now(), limit)
		if !errors.Is(err, ErrWorkQuery) {
			t.Fatalf("invalid prune accepted: %v", err)
		}
	}

	canceled, stop := context.WithCancel(context.Background())
	stop()

	_, err = b.PruneMessages(canceled, time.Now(), 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("prune ignored deadline: %v", err)
	}
}

func deliveryCapacity(t *testing.T) {
	b, db := boundedFixture(t)
	for index := range 100 {
		durableMessage(t, b, fmt.Sprintf("retained-%d", index))
	}

	_, err := db.Exec(`ALTER TABLE pubsub_capacity_slots ADD CONSTRAINT retain_reservation CHECK(message_id IS NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = b.PruneMessages(ctx, time.Now().Add(time.Hour), 1)
	if err == nil {
		t.Fatal("capacity release failure did not roll back deletion")
	}

	var messages, slots int

	err = db.QueryRow(`SELECT (SELECT count(*) FROM pubsub_messages),(SELECT count(*) FROM pubsub_capacity_slots WHERE message_id IS NOT NULL)`).Scan(&messages, &slots)
	if err != nil || messages != 100 || slots != 100 {
		t.Fatalf("prune rollback messages=%d slots=%d error=%v", messages, slots, err)
	}

	_, err = db.Exec(`ALTER TABLE pubsub_capacity_slots DROP CONSTRAINT retain_reservation`)
	if err != nil {
		t.Fatal(err)
	}

	var (
		admitted atomic.Int32
		pruned   int64
		group    sync.WaitGroup
	)

	start := make(chan struct{})

	group.Go(func() {
		<-start

		count, err := b.PruneMessages(ctx, time.Now().Add(time.Hour), 10)
		if err != nil {
			t.Error(err)
		}

		pruned = count
	})

	for index := range 20 {
		group.Go(func() {
			<-start

			env := pubsub.NewEnvelope("events", "new")
			env.ID = fmt.Sprintf("new-%d", index)

			err := b.Publish(ctx, "events", env)
			if err == nil {
				admitted.Add(1)
			} else if !errors.Is(err, ErrCapacity) {
				t.Error(err)
			}
		})
	}

	close(start)
	group.Wait()

	err = db.QueryRow(`SELECT (SELECT count(*) FROM pubsub_messages),(SELECT count(*) FROM pubsub_capacity_slots WHERE message_id IS NOT NULL)`).Scan(&messages, &slots)
	if err != nil || pruned != 10 || messages != 100-int(pruned)+int(admitted.Load()) || messages > 100 || slots != messages {
		t.Fatalf("concurrent capacity messages=%d slots=%d removed=%d admitted=%d error=%v", messages, slots, pruned, admitted.Load(), err)
	}

	if messages < 100 {
		durableMessage(t, b, "after-prune")
	}
}

func deliveryRetirement(t *testing.T) {
	b, db := boundedFixture(t)
	opts := durableOptions()
	opts.Policy.MaxAttempts = 2
	durableSub(t, b, "worker", opts, nil)
	active := durableMessage(t, b, "active")

	old, _, _, err := b.claimDelivery(context.Background(), "worker", active)
	if err != nil {
		t.Fatal(err)
	}

	waiting := durableMessage(t, b, "waiting")
	finishDelivery(t, b, "worker", waiting, "handler_error")
	durableMessage(t, b, "pending")
	succeeded := durableMessage(t, b, "succeeded")
	finishDelivery(t, b, "worker", succeeded, "")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = b.RetireSubscriber(ctx, "worker", RetireOptions{})
	if !errors.Is(err, ErrPendingDelivery) {
		t.Fatalf("unfinished work silently retired: %v", err)
	}

	err = b.RetireSubscriber(ctx, "worker", RetireOptions{DiscardPending: true})
	if err != nil {
		t.Fatal(err)
	}

	err = b.completeDelivery(ctx, old, "")
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("retired owner acknowledged: %v", err)
	}

	err = b.registerDurable(ctx, "worker", "events", opts, false)
	if !errors.Is(err, ErrSubscriberRetired) {
		t.Fatalf("retired identity reopened: %v", err)
	}

	err = b.ConfigureDurableSubscription(ctx, "worker", "events", opts)
	if !errors.Is(err, ErrSubscriberRetired) {
		t.Fatalf("retirement bypassed by configuration: %v", err)
	}

	err = b.RetireSubscriber(ctx, "worker", RetireOptions{})
	if err != nil {
		t.Fatal(err)
	}

	row := inspectedDelivery(t, b, "worker", active)
	if row.State != "failed" || row.FailureCode != "subscriber_retired" || row.Attempt != old.attempt {
		t.Fatalf("retirement invented history: state=%s code=%s attempt=%d", row.State, row.FailureCode, row.Attempt)
	}

	durableMessage(t, b, "future")

	rows, err := b.ListDeliveries(ctx, DeliveryQuery{SubscriberID: "worker"})
	if err != nil || len(rows) != 3 {
		t.Fatalf("retirement admitted pending/future work: rows=%d error=%v", len(rows), err)
	}

	count, err := b.PruneMessages(ctx, time.Now().Add(time.Hour), 1000)
	if err != nil || count != 5 {
		t.Fatalf("explicitly retired work could not expire: %d %v", count, err)
	}

	var retained int

	err = db.QueryRow(`SELECT count(*) FROM pubsub_subscriptions WHERE id='worker' AND retired_at IS NOT NULL`).Scan(&retained)
	if err != nil || retained != 1 {
		t.Fatalf("retired identity binding lost: %d %v", retained, err)
	}
	// Retirement without discard also succeeds when all entitled work is terminal.
	durableSub(t, b, "terminal-only", opts, nil)

	err = b.RetireSubscriber(ctx, "terminal-only", RetireOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func deliveryReplayPrune(t *testing.T) {
	b, _ := boundedFixture(t)
	opts := durableOptions()
	opts.Policy.MaxAttempts = 1
	durableSub(t, b, "worker", opts, nil)

	for index := range 12 {
		id := durableMessage(t, b, fmt.Sprintf("race-%d", index))
		finishDelivery(t, b, "worker", id, "handler_error")
		row := inspectedDelivery(t, b, "worker", id)

		var (
			replayErr, pruneErr error
			count               int64
			group               sync.WaitGroup
		)

		start := make(chan struct{})

		group.Go(func() {
			<-start

			replayErr = b.ReplayDelivery(context.Background(), ReplayRequest{"worker", id, row.Attempt, row.UpdatedAt, 1, ReplayOperator})
		})
		group.Go(func() {
			<-start

			count, pruneErr = b.PruneMessages(context.Background(), time.Now().Add(time.Hour), 1000)
		})
		close(start)
		group.Wait()

		if pruneErr != nil {
			t.Fatal(pruneErr)
		}

		if replayErr == nil {
			if count != 0 {
				t.Fatal("successful replay was pruned")
			}

			current := inspectedDelivery(t, b, "worker", id)
			if current.State != "retry_wait" {
				t.Fatalf("replayed work state=%s", current.State)
			}
		} else if !errors.Is(replayErr, ErrReplayConflict) || count != 1 {
			t.Fatalf("prune/replay outcome count=%d replay=%v", count, replayErr)
		}
	}
}

func deliveryLegacyRetention(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)
	b := deliveryBroker(t, db, 2)

	var calls []string

	legacy := deliverySubscriber(t, b, "legacy", "events", &calls)
	insertDelivery(t, db, "done", "events")
	pollDelivery(t, b, legacy)
	insertDelivery(t, db, "pending", "events")
	insertDelivery(t, db, "unbounded", "unbounded")
	execDeliverySQL(t, db, `ALTER TABLE pubsub_subscriptions DROP COLUMN retired_at`)

	err := b.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	err = b.AdmitBoundedTopics(context.Background(), BoundedConfig{Capacity: 100, Topics: []string{"events"}})
	if err != nil {
		t.Fatal(err)
	}

	count, err := b.PruneMessages(context.Background(), time.Now().Add(time.Hour), 1000)
	if err != nil || count != 1 {
		t.Fatalf("upgrade lost pending entitlement: removed=%d error=%v", count, err)
	}

	err = b.ConfigureDurableSubscription(context.Background(), "legacy", "events", durableOptions())
	if err != nil {
		t.Fatal(err)
	}

	var pending int64

	err = db.QueryRow(`SELECT id FROM pubsub_messages WHERE message_id='pending'`).Scan(&pending)
	if err != nil {
		t.Fatal(err)
	}

	finishDelivery(t, b, "legacy", pending, "")

	count, err = b.PruneMessages(context.Background(), time.Now().Add(time.Hour), 1000)
	if err != nil || count != 1 {
		t.Fatalf("upgraded terminal work retained: %d %v", count, err)
	}

	var unbounded int

	err = db.QueryRow(`SELECT count(*) FROM pubsub_messages WHERE message_id='unbounded' AND NOT bounded`).Scan(&unbounded)
	if err != nil || unbounded != 1 {
		t.Fatalf("bounded prune removed legacy storage: %d %v", unbounded, err)
	}
}
