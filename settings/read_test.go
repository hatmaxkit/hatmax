// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package settings

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

type readCase struct {
	name         string
	typeOf       Type
	defaultRaw   string
	defaultValue any
	storedRaw    string
	storedValue  any
	zero         any
	get          func(*Service, context.Context, string) (any, error)
}

func readCases() []readCase {
	return []readCase{
		{
			name: "string", typeOf: String, defaultRaw: "fallback", defaultValue: "fallback",
			storedRaw: "stored", storedValue: "stored", zero: "",
			get: func(s *Service, ctx context.Context, key string) (any, error) { return s.GetString(ctx, key) },
		},
		{
			name: "int", typeOf: Int, defaultRaw: "50", defaultValue: 50,
			storedRaw: "100", storedValue: 100, zero: 0,
			get: func(s *Service, ctx context.Context, key string) (any, error) { return s.GetInt(ctx, key) },
		},
		{
			name: "bool", typeOf: Bool, defaultRaw: "true", defaultValue: true,
			storedRaw: "false", storedValue: false, zero: false,
			get: func(s *Service, ctx context.Context, key string) (any, error) { return s.GetBool(ctx, key) },
		},
	}
}

// Only a not-found result may select a default. Explicit emptiness and malformed
// stored values keep their meaning and cannot silently restore a registered value.
func TestReadState(t *testing.T) {
	for _, getter := range readCases() {
		for _, state := range []string{"absent", "wrapped absent", "present", "empty", "invalid", "zero"} {
			t.Run(getter.name+"/"+state, func(t *testing.T) {
				registry := NewRegistry()
				registry.Register(Schema{Key: "key", Type: getter.typeOf, Default: getter.defaultRaw})

				store := newMemStore()
				want := getter.defaultValue

				var wantErr error

				switch state {
				case "wrapped absent":
					store.err = fmt.Errorf("adapter lookup: %w", ErrNotFound)
				case "present":
					store.data["key"] = getter.storedRaw
					want = getter.storedValue
				case "empty", "invalid":
					store.data["key"] = ""
					if state == "invalid" {
						store.data["key"] = "invalid value"
					}

					want = store.data["key"]
					if getter.typeOf != String {
						want = getter.zero
						wantErr = strconv.ErrSyntax
					}
				case "zero":
					store.data["key"] = fmt.Sprint(getter.zero)
					want = getter.zero
				}

				before := len(store.data)
				service := NewService(registry, store)

				got, err := getter.get(service, context.Background(), "key")
				if got != want || !errors.Is(err, wantErr) {
					t.Fatalf("got %v, %v; want %v, %v", got, err, want, wantErr)
				}

				if len(store.data) != before {
					t.Fatal("read persisted a fallback value")
				}
			})
		}
	}
}

// A healthy registered default must not hide any non-absence read error. Error
// identity survives unchanged, including cancellation, deadline, and wrapped errors.
func TestReadFailure(t *testing.T) {
	unavailable := errors.New("store unavailable")

	for _, getter := range readCases() {
		for _, failure := range []struct {
			name string
			err  error
		}{
			{"unavailable", unavailable},
			{"wrapped unavailable", fmt.Errorf("lookup: %w", unavailable)},
			{"canceled", context.Canceled},
			{"expired", context.DeadlineExceeded},
			{"untranslated absence", errors.New("setting not found")},
		} {
			t.Run(getter.name+"/"+failure.name, func(t *testing.T) {
				registry := NewRegistry()
				registry.Register(Schema{Key: "key", Type: getter.typeOf, Default: getter.defaultRaw})

				store := newMemStore()
				store.data["key"] = getter.storedRaw
				store.err = failure.err
				service := NewService(registry, store)

				got, err := getter.get(service, context.Background(), "key")
				if got != getter.zero || err != failure.err {
					t.Fatalf("got %v, %v; want zero value and original error %v", got, err, failure.err)
				}
			})
		}
	}
}

// Defaults are parsed only for absence. Missing unregistered keys or empty
// defaults retain zero-value behavior; a bad default cannot replace stored data.
func TestReadDefault(t *testing.T) {
	for _, getter := range readCases() {
		for _, state := range []string{"unregistered", "empty default", "invalid default", "stored over bad default"} {
			t.Run(getter.name+"/"+state, func(t *testing.T) {
				registry := NewRegistry()
				store := newMemStore()
				want := getter.zero

				var wantErr error

				if state != "unregistered" {
					defaultRaw := ""
					if state != "empty default" {
						defaultRaw = "invalid value"
						want = defaultRaw

						if getter.typeOf != String {
							want = getter.zero
							wantErr = strconv.ErrSyntax
						}
					}

					registry.Register(Schema{Key: "key", Type: getter.typeOf, Default: defaultRaw})
				}

				if state == "stored over bad default" {
					store.data["key"] = getter.storedRaw
					want = getter.storedValue
					wantErr = nil
				}

				service := NewService(registry, store)

				got, err := getter.get(service, context.Background(), "key")
				if got != want || !errors.Is(err, wantErr) {
					t.Fatalf("got %v, %v; want %v, %v", got, err, want, wantErr)
				}
			})
		}
	}
}

// Exercise the actual caller context through a conforming adapter, not just
// synthesized context errors. Even a present value cannot mask cancellation.
func TestReadContext(t *testing.T) {
	for _, getter := range readCases() {
		for _, expired := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/expired %t", getter.name, expired), func(t *testing.T) {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer cancel()

				wantErr := context.DeadlineExceeded

				if !expired {
					ctx, cancel = context.WithCancel(context.Background())
					cancel()

					wantErr = context.Canceled
				}

				registry := NewRegistry()
				registry.Register(Schema{Key: "key", Type: getter.typeOf, Default: getter.defaultRaw})

				store := newMemStore()
				store.data["key"] = getter.storedRaw
				service := NewService(registry, store)

				got, err := getter.get(service, ctx, "key")
				if got != getter.zero || !errors.Is(err, wantErr) {
					t.Fatalf("got %v, %v; want zero value and %v", got, err, wantErr)
				}
			})
		}
	}
}
