// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"hatmax.adrianpk.com/pubsub"
	"hatmax.adrianpk.com/testhelper"
)

// TestRetryBatch keeps only the failed row pending, regardless of its position.
// Successful later rows must not be repeated after a poll or named restart.
func TestRetryBatch(t *testing.T) {
	cases := []struct {
		name   string
		failed string
		acked  []string
		offset int64
	}{
		{"first", "first", []string{"middle", "last"}, 3},
		{"middle", "middle", []string{"first", "last"}, 3},
		{"last", "last", []string{"first", "middle"}, 2},
	}

	for _, tc := range cases {
		for _, restart := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/restart %t", tc.name, restart), func(t *testing.T) {
				db, _, cleanup := testhelper.SetupTestDB(t)
				defer cleanup()

				broker := deliveryBroker(t, db, 3)

				var calls []string

				sub := deliverySubscriber(t, broker, "worker", "events", &calls)
				failed := false
				sub.handler = func(_ context.Context, env pubsub.Envelope) error {
					calls = append(calls, env.ID)
					if env.ID == tc.failed && !failed {
						failed = true

						return errors.New("transient failure")
					}

					return nil
				}

				for _, id := range []string{"first", "middle", "last"} {
					insertDelivery(t, db, id, "events")
				}

				pollDelivery(t, broker, sub)
				assertRetryState(t, db, sub, tc.acked, tc.offset)

				if restart {
					err := broker.Close()
					if err != nil {
						t.Fatal(err)
					}

					broker = deliveryBroker(t, db, 3)
					sub = deliverySubscriber(t, broker, "worker", "events", &calls)
				}

				pollDelivery(t, broker, sub)
				pollDelivery(t, broker, sub)

				want := []string{"first", "middle", "last", tc.failed}
				if !reflect.DeepEqual(calls, want) {
					t.Fatalf("attempts %v, want %v", calls, want)
				}

				assertRetryState(t, db, sub, []string{"first", "middle", "last"}, 3)
			})
		}
	}
}

// TestRetryBounds makes persistent failures observable without an inner retry
// loop. Failed rows stay pending and may fill the next bounded batch.
func TestRetryBounds(t *testing.T) {
	for _, size := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("batch %d", size), func(t *testing.T) {
			db, _, cleanup := testhelper.SetupTestDB(t)
			defer cleanup()

			broker := deliveryBroker(t, db, size)

			var calls []string

			sub := deliverySubscriber(t, broker, "worker", "events", &calls)
			sub.handler = func(_ context.Context, env pubsub.Envelope) error {
				calls = append(calls, env.ID)

				return errors.New("persistent failure")
			}

			for i := 0; i < 7; i++ {
				insertDelivery(t, db, fmt.Sprintf("event-%d", i), "events")
			}

			for poll := 0; poll < 3; poll++ {
				before := len(calls)

				pollDelivery(t, broker, sub)

				if len(calls)-before != size {
					t.Fatalf("one poll attempted %d rows, want %d", len(calls)-before, size)
				}

				for i, id := range calls[before:] {
					if want := fmt.Sprintf("event-%d", i); id != want {
						t.Fatalf("attempted %s, want pending %s", id, want)
					}
				}

				assertRetryState(t, db, sub, nil, 0)
			}
		})
	}
}

// TestRetryFanout ensures one subscriber's failure cannot acknowledge another's
// delivery or repeat a healthy subscriber's completed work.
func TestRetryFanout(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := deliveryBroker(t, db, 2)

	var failing, healthy []string

	bad := deliverySubscriber(t, broker, "failing", "events", &failing)
	good := deliverySubscriber(t, broker, "healthy", "events", &healthy)
	bad.handler = func(_ context.Context, env pubsub.Envelope) error {
		failing = append(failing, env.ID)

		return errors.New("failure")
	}

	insertDelivery(t, db, "event", "events")

	for poll := 0; poll < 2; poll++ {
		pollDelivery(t, broker, bad)
		pollDelivery(t, broker, good)
	}

	if !reflect.DeepEqual(failing, []string{"event", "event"}) ||
		!reflect.DeepEqual(healthy, []string{"event"}) {
		t.Fatalf("failing attempts %v, healthy attempts %v", failing, healthy)
	}

	assertRetryState(t, db, bad, nil, 0)
	assertRetryState(t, db, good, []string{"event"}, 1)
}

// TestInvalidPayload retains a corrupt payload without invoking its handler.
// A valid later row can complete while the corrupt row awaits repair.
func TestInvalidPayload(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := deliveryBroker(t, db, 2)

	var calls []string

	sub := deliverySubscriber(t, broker, "worker", "events", &calls)
	insertDelivery(t, db, "broken", "events")
	insertDelivery(t, db, "valid", "events")
	execDeliverySQL(t, db, `UPDATE pubsub_messages SET payload = 'invalid'::bytea
		WHERE message_id = 'broken'`)
	pollDelivery(t, broker, sub)
	pollDelivery(t, broker, sub)
	assertRetryState(t, db, sub, []string{"valid"}, 2)

	if !reflect.DeepEqual(calls, []string{"valid"}) {
		t.Fatalf("invalid payload invoked handler or valid row repeated: %v", calls)
	}

	execDeliverySQL(t, db, `UPDATE pubsub_messages SET payload = '"repaired"'::bytea
		WHERE message_id = 'broken'`)
	pollDelivery(t, broker, sub)
	pollDelivery(t, broker, sub)
	assertRetryState(t, db, sub, []string{"broken", "valid"}, 2)

	if !reflect.DeepEqual(calls, []string{"valid", "broken"}) {
		t.Fatalf("repaired payload was lost or repeated: %v", calls)
	}
}

// TestRetryLoop uses real polling to check the post-completion delay and Close
// cancellation. A slow failure must not accumulate an immediately ready tick.
func TestRetryLoop(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	const interval = 20 * time.Millisecond

	broker := NewBroker(wrapDB(db), Config{PollInterval: interval, BatchSize: 1}, testhelper.TestLogger())
	ctx := context.Background()

	err := broker.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	started := make(chan time.Time, 2)
	completed := make(chan time.Time, 1)
	attempt := 0

	err = broker.Subscribe(ctx, "events", func(ctx context.Context, _ pubsub.Envelope) error {
		attempt++

		started <- time.Now()

		if attempt == 1 {
			delay := time.NewTimer(3 * interval)
			defer delay.Stop()

			select {
			case <-delay.C:
			case <-ctx.Done():
				return ctx.Err()
			}

			completed <- time.Now()

			return errors.New("slow failure")
		}

		<-ctx.Done()

		return ctx.Err()
	}, pubsub.SubscribeOptions{SubscriberID: "worker"})
	if err != nil {
		t.Fatal(err)
	}

	insertDelivery(t, db, "event", "events")
	waitRetryTime(t, started)
	finished := waitRetryTime(t, completed)
	retried := waitRetryTime(t, started)

	if delay := retried.Sub(finished); delay < interval {
		t.Fatalf("retried after %s, want at least %s", delay, interval)
	}

	err = broker.Close()
	if err != nil {
		t.Fatal(err)
	}

	if attempt != 2 || len(started) != 0 {
		t.Fatalf("polling continued after cancellation: %d attempts", attempt)
	}

	assertRetryState(t, db, broker.subscriptions["worker"], nil, 0)
}

func waitRetryTime(t *testing.T, events <-chan time.Time) time.Time {
	t.Helper()

	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()

	select {
	case event := <-events:
		return event
	case <-deadline.C:
		t.Fatal("polling event timed out")

		return time.Time{}
	}
}

func assertRetryState(t *testing.T, db *sql.DB, sub *subscription, want []string, offset int64) {
	t.Helper()

	rows, err := db.Query(`SELECT m.message_id FROM pubsub_acknowledgements a
		JOIN pubsub_messages m ON m.id = a.message_id
		WHERE a.subscriber_id = $1 ORDER BY m.id`, sub.id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got []string

	for rows.Next() {
		var id string

		err = rows.Scan(&id)
		if err != nil {
			t.Fatal(err)
		}

		got = append(got, id)
	}

	err = rows.Err()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("acknowledged %v, want %v", got, want)
	}

	var stored int64

	err = db.QueryRow(`SELECT last_message_id FROM pubsub_subscriptions WHERE id = $1`, sub.id).Scan(&stored)
	if err != nil {
		t.Fatal(err)
	}

	if stored != offset || sub.lastOffset != offset {
		t.Fatalf("progress database=%d memory=%d, want %d", stored, sub.lastOffset, offset)
	}
}
