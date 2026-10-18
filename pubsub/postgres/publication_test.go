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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/pubsub"
	"hatmax.adrianpk.com/testhelper"
)

// A caller's transaction owns both domain state and publication, while bounded
// identities, retained slots and canonical envelopes survive concurrent writers.
func TestBoundedPublication(t *testing.T) {
	t.Run("domain transaction", boundedTransaction)
	t.Run("concurrent capacity", boundedCapacity)
	t.Run("envelope limits", boundedEnvelopeLimits)
	t.Run("topic admission", boundedTopicAdmission)
}

func boundedFixture(t *testing.T) (*Broker, *sql.DB) {
	t.Helper()
	db, _, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)
	db.SetMaxOpenConns(12)
	b := NewBroker(wrapDB(db), Config{PollInterval: time.Hour, BatchSize: 2, Bounded: &BoundedConfig{Topics: []string{"events", "other"}, Capacity: 100}}, testhelper.TestLogger())

	err := b.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = b.Close() })

	return b, db
}

func boundedTransaction(t *testing.T) {
	b, db := boundedFixture(t)
	ctx := context.Background()

	_, err := db.Exec(`CREATE TABLE domain_effects(id TEXT PRIMARY KEY)`)
	if err != nil {
		t.Fatal(err)
	}

	env := pubsub.NewEnvelope("events", json.RawMessage(`{"b":2,"a":{"d":4,"c":3}}`))
	env.ID = "transaction"

	for _, commit := range []bool{false, true} {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}

		_, err = tx.Exec(`INSERT INTO domain_effects VALUES('effect')`)
		if err != nil {
			t.Fatal(err)
		}

		err = b.PublishTx(ctx, tx, "events", env)
		if err != nil {
			t.Fatal(err)
		}

		var visible int

		err = db.QueryRow(`SELECT count(*) FROM pubsub_messages`).Scan(&visible)
		if err != nil || visible != 0 {
			t.Fatalf("uncommitted publication visible=%d error=%v", visible, err)
		}

		if commit {
			err = tx.Commit()
		} else {
			err = tx.Rollback()
		}

		if err != nil {
			t.Fatal(err)
		}

		var domains, messages int

		err = db.QueryRow(`SELECT (SELECT count(*) FROM domain_effects),(SELECT count(*) FROM pubsub_messages)`).Scan(&domains, &messages)

		want := 0
		if commit {
			want = 1
		}

		if err != nil || domains != want || messages != want {
			t.Fatalf("atomic publication: domain=%d messages=%d error=%v", domains, messages, err)
		}
	}

	env.Payload = json.RawMessage(`{"a":{"c":3,"d":4},"b":2}`)
	env.Timestamp = env.Timestamp.Add(time.Hour)

	err = b.Publish(ctx, "events", env)
	if err != nil {
		t.Fatalf("canonical duplicate: %v", err)
	}

	for _, test := range []struct {
		name   string
		change func(*pubsub.Envelope)
	}{
		{"topic", func(e *pubsub.Envelope) { e.Topic = "other" }},
		{"payload", func(e *pubsub.Envelope) { e.Payload = "different" }},
		{"metadata", func(e *pubsub.Envelope) { e.Metadata = map[string]string{"key": "changed"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			other := env
			test.change(&other)

			err := b.Publish(ctx, other.Topic, other)
			if !errors.Is(err, ErrIdentityConflict) {
				t.Fatalf("conflicting identity accepted: %v", err)
			}
		})
	}
}

func boundedCapacity(t *testing.T) {
	b, db := boundedFixture(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	held := pubsub.NewEnvelope("events", "held")
	held.ID = "held"

	err = b.PublishTx(ctx, tx, "events", held)
	if err != nil {
		t.Fatal(err)
	}

	var (
		accepted atomic.Int32
		group    sync.WaitGroup
	)
	for index := range 120 {
		group.Go(func() {
			env := pubsub.NewEnvelope("events", "payload")
			env.ID = fmt.Sprintf("concurrent-%d", index)

			err := b.Publish(ctx, "events", env)
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrCapacity) {
				t.Error(err)
			}
		})
	}

	group.Wait()

	if accepted.Load() != 99 {
		t.Fatalf("accepted %d messages while one slot reserved", accepted.Load())
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	err = b.Publish(ctx, "events", held)
	if err != nil {
		t.Fatalf("rollback leaked capacity: %v", err)
	}

	err = b.Publish(ctx, "events", held)
	if err != nil {
		t.Fatalf("duplicate at capacity: %v", err)
	}

	more := pubsub.NewEnvelope("events", "overflow")

	err = b.Publish(ctx, "events", more)
	if !errors.Is(err, ErrCapacity) {
		t.Fatalf("capacity overflow accepted: %v", err)
	}

	var messages, slots int

	err = db.QueryRow(`SELECT (SELECT count(*) FROM pubsub_messages WHERE bounded),(SELECT count(*) FROM pubsub_capacity_slots WHERE message_id IS NOT NULL)`).Scan(&messages, &slots)
	if err != nil || messages != 100 || slots != 100 {
		t.Fatalf("capacity messages=%d slots=%d error=%v", messages, slots, err)
	}
}

func boundedEnvelopeLimits(t *testing.T) {
	b, _ := boundedFixture(t)
	for _, test := range []struct {
		name   string
		change func(*pubsub.Envelope)
		valid  bool
	}{
		{"maximum id", func(e *pubsub.Envelope) { e.ID = strings.Repeat("a", 128) }, true},
		{"long id", func(e *pubsub.Envelope) { e.ID = strings.Repeat("a", 129) }, false},
		{"empty id", func(e *pubsub.Envelope) { e.ID = "" }, false},
		{"display id", func(e *pubsub.Envelope) { e.ID = "arbitrary display text" }, false},
		{"unicode id", func(e *pubsub.Envelope) { e.ID = "caf\u00e9" }, false},
		{"bad first byte", func(e *pubsub.Envelope) { e.ID = "/bad" }, false},
		{"topic disagreement", func(e *pubsub.Envelope) { e.Topic = "other" }, false},
		{"maximum payload", func(e *pubsub.Envelope) { e.Payload = strings.Repeat("x", 65534) }, true},
		{"large payload", func(e *pubsub.Envelope) { e.Payload = strings.Repeat("x", 65535) }, false},
		{"invalid json", func(e *pubsub.Envelope) { e.Payload = json.RawMessage(`broken`) }, false},
		{"unencodable payload", func(e *pubsub.Envelope) { e.Payload = make(chan int) }, false},
		{"maximum metadata", func(e *pubsub.Envelope) { e.Metadata = map[string]string{"x": strings.Repeat("x", 8183)} }, true},
		{"large metadata", func(e *pubsub.Envelope) { e.Metadata = map[string]string{"x": strings.Repeat("x", 8192)} }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := pubsub.NewEnvelope("events", "valid")
			test.change(&env)

			err := b.Publish(context.Background(), "events", env)
			if (err == nil) != test.valid {
				t.Fatalf("admission validity=%t error=%v", test.valid, err)
			}
		})
	}
}

func boundedTopicAdmission(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)
	b := deliveryBroker(t, db, 2)
	env := pubsub.NewEnvelope("legacy", "payload")
	env.ID = "retained"

	err := b.Publish(context.Background(), "legacy", env)
	if err != nil {
		t.Fatal(err)
	}

	cfg := BoundedConfig{Capacity: 100, Topics: []string{"legacy"}}

	err = b.configureBounded(context.Background(), cfg, false)
	if !errors.Is(err, ErrAdmissionUpgrade) {
		t.Fatalf("legacy topic silently admitted: %v", err)
	}

	err = b.AdmitBoundedTopics(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = b.Publish(context.Background(), "legacy", env)
	if err != nil {
		t.Fatalf("audited duplicate: %v", err)
	}

	for _, cfg := range []BoundedConfig{{Capacity: 99, Topics: []string{"x"}}, {Capacity: 100001, Topics: []string{"x"}}, {Capacity: 100, Topics: []string{"bad topic"}}, {Capacity: 100, Topics: []string{"x", "x"}}} {
		err = b.AdmitBoundedTopics(context.Background(), cfg)
		if err == nil {
			t.Fatal("invalid admission configuration accepted")
		}
	}

	topics := make([]string, 127)
	for i := range topics {
		topics[i] = fmt.Sprintf("topic-%d", i)
	}

	err = b.AdmitBoundedTopics(context.Background(), BoundedConfig{Capacity: 100, Topics: topics})
	if err != nil {
		t.Fatal(err)
	}

	err = b.AdmitBoundedTopics(context.Background(), BoundedConfig{Capacity: 100, Topics: []string{"overflow"}})
	if !errors.Is(err, ErrAdmissionLimit) {
		t.Fatalf("topic count exceeded: %v", err)
	}

	err = b.AdmitBoundedTopics(context.Background(), BoundedConfig{Capacity: 101, Topics: []string{"legacy"}})
	if !errors.Is(err, ErrPolicyConflict) {
		t.Fatalf("capacity silently changed: %v", err)
	}
}
