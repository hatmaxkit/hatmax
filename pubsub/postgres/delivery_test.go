// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"hatmax.adrianpk.com/pubsub"
	"hatmax.adrianpk.com/testhelper"
)

// TestLateCommit controls commit order without timing sleeps. A lower-ID
// transaction stays invisible while the higher row is delivered and persisted.
func TestLateCommit(t *testing.T) {
	for _, restart := range []bool{false, true} {
		name := "same broker"
		if restart {
			name = "after restart"
		}

		t.Run(name, func(t *testing.T) {
			db, _, cleanup := testhelper.SetupTestDB(t)
			defer cleanup()

			broker := deliveryBroker(t, db, 1)

			var received []string

			sub := deliverySubscriber(t, broker, "worker", "events", &received)

			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()

			insertDelivery(t, tx, "late", "events")
			insertDelivery(t, db, "early", "events")
			pollDelivery(t, broker, sub)

			if !reflect.DeepEqual(received, []string{"early"}) {
				t.Fatalf("first poll delivered %v", received)
			}

			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}

			if restart {
				broker = deliveryBroker(t, db, 1)
				sub = deliverySubscriber(t, broker, "worker", "events", &received)
			}

			pollDelivery(t, broker, sub)
			pollDelivery(t, broker, sub)

			if !reflect.DeepEqual(received, []string{"early", "late"}) {
				t.Fatalf("late committed row was lost or repeated: %v", received)
			}
		})
	}
}

// TestSubscriptionSnapshot skips visible history but not an in-flight lower ID.
func TestSubscriptionSnapshot(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := deliveryBroker(t, db, 2)

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	insertDelivery(t, tx, "late", "events")
	insertDelivery(t, db, "history", "events")

	var received []string

	sub := deliverySubscriber(t, broker, "worker", "events", &received)
	pollDelivery(t, broker, sub)

	if len(received) != 0 {
		t.Fatalf("new subscriber replayed visible history: %v", received)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatal(err)
	}

	insertDelivery(t, db, "new", "events")
	pollDelivery(t, broker, sub)
	pollDelivery(t, broker, sub)

	if !reflect.DeepEqual(received, []string{"late", "new"}) {
		t.Fatalf("snapshot boundary lost or replayed messages: %v", received)
	}
}

// TestDeliveryBatches checks finite batch bounds, topic isolation, and fan-out.
func TestDeliveryBatches(t *testing.T) {
	for _, size := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("batch %d", size), func(t *testing.T) {
			db, _, cleanup := testhelper.SetupTestDB(t)
			defer cleanup()

			broker := deliveryBroker(t, db, size)

			var first, second []string

			subs := []*subscription{
				deliverySubscriber(t, broker, "first", "events", &first),
				deliverySubscriber(t, broker, "second", "events", &second),
			}

			want := make([]string, 7)
			for i := range want {
				want[i] = fmt.Sprintf("event-%d", i)
				insertDelivery(t, db, want[i], "events")
			}

			insertDelivery(t, db, "other-topic", "other")

			for index, sub := range subs {
				received := &first
				if index == 1 {
					received = &second
				}

				for poll := 0; poll < (len(want)+size-1)/size+1; poll++ {
					before := len(*received)

					pollDelivery(t, broker, sub)

					if got := len(*received) - before; got > size {
						t.Fatalf("one poll delivered %d messages with batch size %d", got, size)
					}
				}

				if !reflect.DeepEqual(*received, want) {
					t.Fatalf("subscriber %s received %v, want %v", sub.id, *received, want)
				}
			}
		})
	}
}

// TestAckRollback injects a persistence failure after handlers run. Neither the
// exact acknowledgements nor the diagnostic offset may commit independently.
func TestAckRollback(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := deliveryBroker(t, db, 2)

	var received []string

	sub := deliverySubscriber(t, broker, "worker", "events", &received)
	insertDelivery(t, db, "first", "events")
	insertDelivery(t, db, "second", "events")
	execDeliverySQL(t, db, `ALTER TABLE pubsub_subscriptions
		ADD CONSTRAINT fail_progress CHECK (last_message_id = 0)`)

	err := broker.pollMessages(context.Background(), sub)
	if err == nil {
		t.Fatal("expected diagnostic progress failure")
	}

	var acknowledgements, offset int64

	err = db.QueryRow(`SELECT COUNT(*) FROM pubsub_acknowledgements`).Scan(&acknowledgements)
	if err != nil {
		t.Fatal(err)
	}

	err = db.QueryRow(`SELECT last_message_id FROM pubsub_subscriptions WHERE id = 'worker'`).Scan(&offset)
	if err != nil {
		t.Fatal(err)
	}

	if acknowledgements != 0 || offset != 0 || sub.lastOffset != 0 {
		t.Fatalf("failed statement committed acknowledgements=%d, offset=%d, memory=%d",
			acknowledgements, offset, sub.lastOffset)
	}

	execDeliverySQL(t, db, `ALTER TABLE pubsub_subscriptions DROP CONSTRAINT fail_progress`)
	pollDelivery(t, broker, sub)
	pollDelivery(t, broker, sub)

	if !reflect.DeepEqual(received, []string{"first", "second", "first", "second"}) {
		t.Fatalf("uncommitted acknowledgements did not redeliver: %v", received)
	}
}

// TestLegacyDelivery migrates the old scalar cursor once without replaying its
// visible acknowledged history or discarding rows above it.
func TestLegacyDelivery(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	execDeliverySQL(t, db, legacyDeliverySchema)
	insertDelivery(t, db, "processed", "events")
	insertDelivery(t, db, "pending", "events")
	execDeliverySQL(t, db, `INSERT INTO pubsub_subscriptions (id, topic, last_message_id)
		VALUES ('worker', 'events', 1)`)

	broker := deliveryBroker(t, db, 2)

	var received []string

	sub := deliverySubscriber(t, broker, "worker", "events", &received)
	pollDelivery(t, broker, sub)
	broker = deliveryBroker(t, db, 2)
	sub = deliverySubscriber(t, broker, "worker", "events", &received)
	pollDelivery(t, broker, sub)

	if !reflect.DeepEqual(received, []string{"pending"}) {
		t.Fatalf("legacy migration lost or replayed messages: %v", received)
	}
}

// TestMigrationRollback checks that a failed legacy backfill cannot leave a
// subscription marked initialized or commit the new schema column.
func TestMigrationRollback(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	execDeliverySQL(t, db, legacyDeliverySchema)
	insertDelivery(t, db, "processed", "events")
	execDeliverySQL(t, db, `INSERT INTO pubsub_subscriptions (id, topic, last_message_id)
		VALUES ('worker', 'events', 1)`)
	execDeliverySQL(t, db, `CREATE TABLE pubsub_acknowledgements (
		subscriber_id TEXT CHECK (subscriber_id <> 'worker'), message_id BIGINT,
		PRIMARY KEY (subscriber_id, message_id))`)

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())

	err := broker.Start(context.Background())
	if err == nil {
		t.Fatal("expected migration constraint failure")
	}

	var columns int

	err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'pubsub_subscriptions'
		AND column_name = 'acknowledgements_initialized'`).Scan(&columns)
	if err != nil {
		t.Fatal(err)
	}

	if columns != 0 {
		t.Fatal("failed migration committed its new schema column")
	}
}

// This fixture retains the pre-acknowledgement schema for upgrade tests.
const legacyDeliverySchema = `
	CREATE TABLE pubsub_messages (
		id BIGSERIAL PRIMARY KEY, message_id TEXT UNIQUE NOT NULL,
		topic TEXT NOT NULL, payload BYTEA NOT NULL, metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE pubsub_subscriptions (
		id TEXT PRIMARY KEY, topic TEXT NOT NULL, last_message_id BIGINT DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

func execDeliverySQL(t *testing.T, db *sql.DB, query string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
}

func deliveryBroker(t *testing.T, db *sql.DB, batchSize int) *Broker {
	t.Helper()

	broker := NewBroker(wrapDB(db), Config{BatchSize: batchSize}, testhelper.TestLogger())

	err := broker.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { broker.Close() })

	return broker
}

// Tests drive the poll boundary directly to keep commit order deterministic.
func deliverySubscriber(t *testing.T, broker *Broker, id, topic string, received *[]string) *subscription {
	t.Helper()

	offset, err := broker.getOrCreateSubscription(context.Background(), id, topic)
	if err != nil {
		t.Fatal(err)
	}

	return &subscription{
		id: id, topic: topic, lastOffset: offset,
		handler: func(_ context.Context, env pubsub.Envelope) error {
			*received = append(*received, env.ID)

			return nil
		},
	}
}

func insertDelivery(t *testing.T, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, id, topic string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO pubsub_messages (message_id, topic, payload, metadata, created_at)
		VALUES ($1, $2, $3, '{}', $4)`, id, topic, []byte(`"payload"`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func pollDelivery(t *testing.T, broker *Broker, sub *subscription) {
	t.Helper()

	err := broker.pollMessages(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
}
