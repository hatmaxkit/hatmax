// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/pubsub"
	"hatmax.adrianpk.com/testhelper"
)

// Real database attempts retain per-subscriber ownership, late commits, fixed
// budgets and safe terminal failures without monopolizing the next poll batch.
func TestBoundedDeliveries(t *testing.T) {
	if schema := os.Getenv("PUBSUB_CRASH_SCHEMA"); schema != "" {
		deliveryCrashWorker(t, schema)

		return
	}

	t.Run("replicas and fenced ack", durableReplicas)
	t.Run("history and late commits", durableHistory)
	t.Run("quarantine and progress", durableQuarantine)
	t.Run("safe failures", durableFailures)
	t.Run("subscription contracts", durableRegistration)
	t.Run("legacy subscription upgrade", durableUpgrade)
	t.Run("shutdown and connection", durableShutdown)
	t.Run("killed worker", durableInterruption)
	t.Run("expired final attempt", durableExhaustion)
	t.Run("source timestamp", durableTimestamp)
}

// Occurrence time survives duplicate publication and leased retry independently
// of the database publication time used for retained-message cleanup.
func durableTimestamp(t *testing.T) {
	b, db := boundedFixture(t)
	ctx := context.Background()

	durableSub(t, b, "timestamps", durableOptions(), nil)

	source := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
	env := pubsub.NewEnvelope("events", "source")
	env.ID, env.Timestamp = "timestamp", source

	err := b.Publish(ctx, "events", env)
	if err != nil {
		t.Fatal(err)
	}

	env.Timestamp = source.Add(time.Minute)

	err = b.Publish(ctx, "events", env)
	if err != nil {
		t.Fatalf("duplicate publication: %v", err)
	}

	var (
		id        int64
		published time.Time
	)

	err = db.QueryRow(`SELECT id,created_at FROM pubsub_messages WHERE message_id=$1`, env.ID).Scan(&id, &published)
	if err != nil || !published.After(source.Add(30*time.Minute)) {
		t.Fatalf("publication time=%s error=%v", published, err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		claim, delivered, code, err := b.claimDelivery(ctx, "timestamps", id)
		if err != nil || code != "" || !delivered.Timestamp.Equal(source) {
			t.Fatalf("attempt=%d source=%s delivered=%s code=%s error=%v", attempt, source, delivered.Timestamp, code, err)
		}

		err = b.completeDelivery(ctx, claim, "handler_error")
		if err != nil {
			t.Fatal(err)
		}

		dueDelivery(t, db, "timestamps", id)
	}

	for _, test := range []struct {
		name      string
		timestamp time.Time
		legacy    bool
	}{
		{"zero timestamp", time.Time{}, false},
		{"pre-upgrade row", source, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := pubsub.NewEnvelope("events", test.name)
			env.Timestamp = test.timestamp

			err := b.Publish(ctx, "events", env)
			if err != nil {
				t.Fatal(err)
			}

			if test.legacy {
				_, err = db.Exec(`UPDATE pubsub_messages SET occurred_at=NULL WHERE message_id=$1`, env.ID)
				if err != nil {
					t.Fatal(err)
				}
			}

			err = db.QueryRow(`SELECT id,created_at FROM pubsub_messages WHERE message_id=$1`, env.ID).Scan(&id, &published)
			if err != nil {
				t.Fatal(err)
			}

			_, delivered, code, err := b.claimDelivery(ctx, "timestamps", id)
			if err != nil || code != "" || !delivered.Timestamp.Equal(published) {
				t.Fatalf("publication=%s delivered=%s code=%s error=%v", published, delivered.Timestamp, code, err)
			}
		})
	}
}

func durableOptions() pubsub.DurableOptions {
	return pubsub.DurableOptions{Policy: pubsub.DefaultDeliveryPolicy(), History: pubsub.RetainedHistory}
}

func durableSub(t *testing.T, b *Broker, id string, opts pubsub.DurableOptions, handler pubsub.Handler) *subscription {
	t.Helper()

	err := b.registerDurable(context.Background(), id, "events", opts, false)
	if err != nil {
		t.Fatal(err)
	}

	return &subscription{id: id, topic: "events", handler: handler, bounded: true}
}

func durableMessage(t *testing.T, b *Broker, id string) int64 {
	t.Helper()

	env := pubsub.NewEnvelope("events", map[string]any{"value": id})
	env.ID = id

	err := b.Publish(context.Background(), "events", env)
	if err != nil {
		t.Fatal(err)
	}

	var row int64

	err = b.db.QueryRow(`SELECT id FROM pubsub_messages WHERE message_id=$1`, id).Scan(&row)
	if err != nil {
		t.Fatal(err)
	}

	return row
}

func expireDelivery(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`UPDATE pubsub_deliveries SET lease_expires_at=clock_timestamp()-interval '1 second' WHERE state='running'`)
	if err != nil {
		t.Fatal(err)
	}
}

func durableReplicas(t *testing.T) {
	b, db := boundedFixture(t)
	second := deliveryBroker(t, db, 1)
	durableSub(t, b, "worker", durableOptions(), nil)
	id := durableMessage(t, b, "event")

	var (
		wins   atomic.Int32
		group  sync.WaitGroup
		winner deliveryClaim
	)

	for index := range 12 {
		group.Go(func() {
			broker := b
			if index%2 == 1 {
				broker = second
			}

			claim, _, _, err := broker.claimDelivery(context.Background(), "worker", id)
			if err == nil {
				winner = claim

				wins.Add(1)
			} else if !errors.Is(err, ErrDeliveryClaim) {
				t.Error(err)
			}
		})
	}

	group.Wait()

	if wins.Load() != 1 {
		t.Fatalf("claim winners=%d", wins.Load())
	}

	err := b.acknowledgeBatch(context.Background(), "worker", []int64{id}, id)
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("unfenced ack accepted: %v", err)
	}

	expireDelivery(t, db)

	err = b.completeDelivery(context.Background(), winner, "")
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("expired ack accepted: %v", err)
	}

	replacement, _, _, err := second.claimDelivery(context.Background(), "worker", id)
	if err != nil || replacement.attempt != 2 || replacement.token == winner.token {
		t.Fatalf("replacement attempt=%d error=%v", replacement.attempt, err)
	}

	err = b.completeDelivery(context.Background(), winner, "")
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("stale ack accepted: %v", err)
	}

	for _, test := range []struct {
		name   string
		change func(*deliveryClaim)
	}{
		{"attempt", func(c *deliveryClaim) { c.attempt++ }},
		{"message", func(c *deliveryClaim) { c.message++ }},
		{"subscriber", func(c *deliveryClaim) { c.subscriber = "missing" }},
		{"token", func(c *deliveryClaim) { c.token = "wrong" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := replacement
			test.change(&bad)

			err := b.completeDelivery(context.Background(), bad, "")
			if !errors.Is(err, ErrStaleDelivery) {
				t.Fatalf("bad ownership accepted: %v", err)
			}
		})
	}

	_, err = db.Exec(`ALTER TABLE pubsub_subscriptions ADD CONSTRAINT refuse_progress CHECK(last_message_id=0)`)
	if err != nil {
		t.Fatal(err)
	}

	err = second.completeDelivery(context.Background(), replacement, "")
	if err == nil {
		t.Fatal("atomic ack unexpectedly passed")
	}

	var (
		state            string
		acknowledgements int
	)

	err = db.QueryRow(`SELECT state,(SELECT count(*) FROM pubsub_acknowledgements) FROM pubsub_deliveries`).Scan(&state, &acknowledgements)
	if err != nil || state != "running" || acknowledgements != 0 {
		t.Fatalf("partial completion state=%s acks=%d error=%v", state, acknowledgements, err)
	}

	_, err = db.Exec(`ALTER TABLE pubsub_subscriptions DROP CONSTRAINT refuse_progress`)
	if err != nil {
		t.Fatal(err)
	}

	err = second.completeDelivery(context.Background(), replacement, "")
	if err != nil {
		t.Fatal(err)
	}

	err = b.completeDelivery(context.Background(), replacement, "handler_error")
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("terminal result overwritten: %v", err)
	}
}

func durableHistory(t *testing.T) {
	for _, history := range []pubsub.HistoryMode{pubsub.CurrentPublications, pubsub.RetainedHistory} {
		t.Run(string(history), func(t *testing.T) {
			b, db := boundedFixture(t)

			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()

			late := pubsub.NewEnvelope("events", "late")
			late.ID = "late"

			err = b.PublishTx(context.Background(), tx, "events", late)
			if err != nil {
				t.Fatal(err)
			}

			durableMessage(t, b, "history")

			opts := durableOptions()
			opts.History = history

			var first, second []string

			handler := func(target *[]string) pubsub.Handler {
				return func(_ context.Context, e pubsub.Envelope) error {
					*target = append(*target, e.ID)

					return nil
				}
			}
			a := durableSub(t, b, "first", opts, handler(&first))
			c := durableSub(t, b, "second", opts, handler(&second))
			pollDelivery(t, b, a)
			pollDelivery(t, b, c)

			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}

			durableMessage(t, b, "new")
			restarted := deliveryBroker(t, db, 2)

			a = durableSub(t, restarted, "first", opts, handler(&first))
			for range 3 {
				pollDelivery(t, restarted, a)
				pollDelivery(t, b, c)
			}

			want := []string{"late", "new"}
			if history == pubsub.RetainedHistory {
				want = []string{"history", "late", "new"}
			}

			if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, want) {
				t.Fatalf("fanout first=%v second=%v want=%v", first, second, want)
			}
		})
	}
}

func durableQuarantine(t *testing.T) {
	b, db := boundedFixture(t)
	b.cfg.BatchSize = 1
	opts := durableOptions()
	opts.Policy.MaxAttempts = 2
	opts.Policy.RetryDelay = time.Second

	var calls []string

	bad := durableSub(t, b, "failing", opts, func(_ context.Context, e pubsub.Envelope) error {
		calls = append(calls, e.ID)
		if e.ID == "bad" {
			return errors.New("sensitive provider error")
		}

		return nil
	})

	var healthy []string

	good := durableSub(t, b, "healthy", opts, func(_ context.Context, e pubsub.Envelope) error {
		healthy = append(healthy, e.ID)

		return nil
	})
	durableMessage(t, b, "bad")
	durableMessage(t, b, "later")
	pollDelivery(t, b, bad)

	changed := opts
	changed.Policy.MaxAttempts = 32
	changed.Policy.RetryDelay = time.Hour

	err := b.ConfigureDurableSubscription(context.Background(), "failing", "events", changed)
	if err != nil {
		t.Fatal(err)
	}

	pollDelivery(t, b, bad)

	_, err = db.Exec(`UPDATE pubsub_deliveries SET retry_at=clock_timestamp()-interval '1 second' WHERE state='retry_wait'`)
	if err != nil {
		t.Fatal(err)
	}

	restarted := deliveryBroker(t, db, 1)

	bad = durableSub(t, restarted, "failing", changed, bad.handler)
	for range 3 {
		pollDelivery(t, restarted, bad)
		pollDelivery(t, b, good)
	}

	if !reflect.DeepEqual(calls, []string{"bad", "later", "bad"}) || !reflect.DeepEqual(healthy, []string{"bad", "later"}) {
		t.Fatalf("quarantine calls=%v healthy=%v", calls, healthy)
	}

	var (
		state, code          string
		attempt, limit, acks int
	)

	err = db.QueryRow(`SELECT d.state,d.failure_code,d.attempt,d.attempt_limit,(SELECT count(*) FROM pubsub_acknowledgements a WHERE a.subscriber_id=d.subscriber_id AND a.message_id=d.message_id)
 FROM pubsub_deliveries d JOIN pubsub_messages m ON m.id=d.message_id WHERE d.subscriber_id='failing' AND m.message_id='bad'`).Scan(&state, &code, &attempt, &limit, &acks)
	if err != nil || state != "failed" || code != "handler_error" || attempt != 2 || limit != 2 || acks != 0 {
		t.Fatalf("quarantine state=%s code=%s attempt=%d limit=%d acks=%d error=%v", state, code, attempt, limit, acks, err)
	}
}

func durableFailures(t *testing.T) {
	for _, test := range []struct {
		name, code string
		handler    pubsub.Handler
		corrupt    string
	}{
		{"panic", "handler_panic", func(context.Context, pubsub.Envelope) error { panic("sensitive panic") }, ""},
		{"deadline", "handler_deadline", func(ctx context.Context, _ pubsub.Envelope) error {
			<-ctx.Done()

			return nil
		}, ""},
		{"payload", "invalid_payload", nil, `UPDATE pubsub_messages SET payload='broken'::bytea`},
		{"metadata", "invalid_metadata", nil, `UPDATE pubsub_messages SET metadata='42'::jsonb`},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, db := boundedFixture(t)
			opts := durableOptions()
			opts.Policy.MaxAttempts = 1
			opts.Policy.HandlerTimeout = time.Second
			calls := 0
			sub := durableSub(t, b, "worker", opts, func(ctx context.Context, e pubsub.Envelope) error {
				calls++

				if test.handler != nil {
					return test.handler(ctx, e)
				}

				return nil
			})
			durableMessage(t, b, "event")

			if test.corrupt != "" {
				_, err := db.Exec(test.corrupt)
				if err != nil {
					t.Fatal(err)
				}
			}

			pollDelivery(t, b, sub)
			pollDelivery(t, b, sub)

			var (
				state, code string
				token       sql.NullString
				acks        int
			)

			err := db.QueryRow(`SELECT state,failure_code,lease_token,(SELECT count(*) FROM pubsub_acknowledgements) FROM pubsub_deliveries`).Scan(&state, &code, &token, &acks)
			if err != nil || state != "failed" || code != test.code || token.Valid || acks != 0 {
				t.Fatalf("classified failure state=%s code=%s token=%t acks=%d error=%v", state, code, token.Valid, acks, err)
			}

			if test.corrupt != "" && calls != 0 {
				t.Fatal("invalid message reached handler")
			}
		})
	}
}

func durableRegistration(t *testing.T) {
	b, _ := boundedFixture(t)
	opts := durableOptions()
	durableSub(t, b, "stable", opts, nil)
	changed := opts

	changed.Policy.MaxAttempts = 1
	for _, test := range []struct {
		name, topic, id string
		options         pubsub.DurableOptions
	}{
		{"topic binding", "other", "stable", opts},
		{"fixed policy", "events", "stable", changed},
		{"missing identity", "events", "", opts},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := b.registerDurable(context.Background(), test.id, test.topic, test.options, false)
			if err == nil {
				t.Fatal("invalid subscription accepted")
			}
		})
	}

	var (
		registered atomic.Int32
		group      sync.WaitGroup
	)
	for index := range 70 {
		group.Go(func() {
			err := b.registerDurable(context.Background(), fmt.Sprintf("subscriber-%d", index), "events", opts, false)
			if err == nil {
				registered.Add(1)
			} else if !errors.Is(err, ErrAdmissionLimit) {
				t.Error(err)
			}
		})
	}

	group.Wait()

	if registered.Load() != 63 {
		t.Fatalf("registered %d beyond existing subscriber", registered.Load())
	}

	_, err := b.getOrCreateSubscription(context.Background(), "legacy", "events")
	if !errors.Is(err, ErrAdmissionUpgrade) {
		t.Fatalf("legacy subscription admitted to bounded topic: %v", err)
	}
}

func durableUpgrade(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	b := deliveryBroker(t, db, 2)

	var calls []string

	legacy := deliverySubscriber(t, b, "worker", "events", &calls)
	insertDelivery(t, db, "succeeded", "events")
	pollDelivery(t, b, legacy)
	insertDelivery(t, db, "pending", "events")

	err := b.AdmitBoundedTopics(context.Background(), BoundedConfig{Capacity: 100, Topics: []string{"events"}})
	if err != nil {
		t.Fatal(err)
	}

	err = b.registerDurable(context.Background(), "worker", "events", durableOptions(), false)
	if !errors.Is(err, ErrSubscriptionConflict) {
		t.Fatalf("legacy subscriber silently upgraded: %v", err)
	}

	err = b.ConfigureDurableSubscription(context.Background(), "worker", "events", durableOptions())
	if err != nil {
		t.Fatal(err)
	}

	upgraded := durableSub(t, b, "worker", durableOptions(), func(_ context.Context, e pubsub.Envelope) error {
		calls = append(calls, e.ID)

		return nil
	})
	for range 2 {
		pollDelivery(t, b, upgraded)
	}

	if !reflect.DeepEqual(calls, []string{"succeeded", "pending"}) {
		t.Fatalf("legacy ack replayed or pending work lost: %v", calls)
	}
}

func durableShutdown(t *testing.T) {
	b, db := boundedFixture(t)
	db.SetMaxOpenConns(1)
	// A handler uses the only connection, proving fetch and claim resources closed.
	sub := durableSub(t, b, "connection", durableOptions(), func(ctx context.Context, _ pubsub.Envelope) error {
		limited, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		var one int

		return db.QueryRowContext(limited, `SELECT 1`).Scan(&one)
	})
	durableMessage(t, b, "connection-event")
	pollDelivery(t, b, sub)
	b.cfg.PollInterval = time.Millisecond
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	opts := durableOptions()

	err := b.Subscribe(context.Background(), "events", func(ctx context.Context, _ pubsub.Envelope) error {
		close(entered)
		<-ctx.Done()
		close(canceled)
		<-release

		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "live", Durable: &opts})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		close(release)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = b.Stop(ctx)
	})

	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler not entered")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = b.Stop(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("outstanding handler reported stopped: %v", err)
	}

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("handler missed cancellation")
	}

	var state string

	err = db.QueryRow(`SELECT state FROM pubsub_deliveries WHERE subscriber_id='live'`).Scan(&state)
	if err != nil || state != "running" {
		t.Fatalf("outstanding handler state=%s error=%v", state, err)
	}

	err = db.Ping()
	if err != nil {
		t.Fatalf("Stop closed caller database: %v", err)
	}
}

func deliveryCrashWorker(t *testing.T, schema string) {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), schema)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	b := deliveryBroker(t, db, 1)

	var id int64

	err = db.QueryRow(`SELECT id FROM pubsub_messages WHERE message_id='event'`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	claim, _, _, err := b.claimDelivery(context.Background(), "worker", id)
	if err != nil {
		t.Fatal(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(struct {
		Attempt int
		Token   string
	}{claim.attempt, claim.token})
	if err != nil {
		t.Fatal(err)
	}

	select {}
}

func durableInterruption(t *testing.T) {
	b, db := boundedFixture(t)
	durableSub(t, b, "worker", durableOptions(), nil)
	id := durableMessage(t, b, "event")

	var schema string

	err := db.QueryRow(`SELECT current_schema()`).Scan(&schema)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBoundedDeliveries$")

	cmd.Env = append(os.Environ(), "PUBSUB_CRASH_SCHEMA="+schema)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	var old struct {
		Attempt int
		Token   string
	}

	err = json.NewDecoder(pipe).Decode(&old)
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Process.Kill()
	if err != nil {
		t.Fatal(err)
	}

	_ = cmd.Wait()

	expireDelivery(t, db)

	fresh, _, _, err := b.claimDelivery(context.Background(), "worker", id)
	if err != nil || fresh.attempt != old.Attempt+1 {
		t.Fatalf("recovered attempt=%d error=%v", fresh.attempt, err)
	}

	err = b.completeDelivery(context.Background(), deliveryClaim{subscriber: "worker", message: id, attempt: old.Attempt, token: old.Token}, "")
	if !errors.Is(err, ErrStaleDelivery) {
		t.Fatalf("dead worker acknowledged: %v", err)
	}

	err = b.completeDelivery(context.Background(), fresh, "")
	if err != nil {
		t.Fatal(err)
	}
}

// Expiry at the retained budget finalizes without another handler or success ack.
func durableExhaustion(t *testing.T) {
	b, db := boundedFixture(t)
	opts := durableOptions()
	opts.Policy.MaxAttempts = 1

	var calls []string

	sub := durableSub(t, b, "worker", opts, func(_ context.Context, e pubsub.Envelope) error {
		calls = append(calls, e.ID)

		return nil
	})
	id := durableMessage(t, b, "lost")
	durableMessage(t, b, "later")

	_, _, _, err := b.claimDelivery(context.Background(), "worker", id)
	if err != nil {
		t.Fatal(err)
	}

	expireDelivery(t, db)
	pollDelivery(t, b, sub)
	pollDelivery(t, b, sub)

	var (
		state, code   string
		attempt, acks int
	)

	err = db.QueryRow(`SELECT d.state,d.failure_code,d.attempt,(SELECT count(*) FROM pubsub_acknowledgements WHERE subscriber_id=d.subscriber_id AND message_id=d.message_id)
 FROM pubsub_deliveries d WHERE subscriber_id='worker' AND message_id=$1`, id).Scan(&state, &code, &attempt, &acks)
	if err != nil || state != "failed" || code != "lease_expired" || attempt != 1 || acks != 0 || !reflect.DeepEqual(calls, []string{"later"}) {
		t.Fatalf("expiry state=%s code=%s attempt=%d acks=%d calls=%v error=%v", state, code, attempt, acks, calls, err)
	}
}
