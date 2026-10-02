// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"context"
	"errors"
	"testing"

	"hatmax.adrianpk.com/settings"
)

// The guide's real adapter must distinguish absence from stored emptiness and
// cancellation, so its service demonstrates the same contract as custom stores.
func TestSettingLookup(t *testing.T) {
	for _, tc := range []struct {
		name         string
		present      bool
		raw          string
		canceled     bool
		storeErr     error
		serviceErr   error
		serviceValue string
	}{
		{name: "absent", storeErr: settings.ErrNotFound, serviceValue: "fallback"},
		{name: "empty", present: true},
		{name: "stored", present: true, raw: "stored", serviceValue: "stored"},
		{name: "canceled absent", canceled: true, storeErr: context.Canceled, serviceErr: context.Canceled},
		{name: "canceled present", present: true, canceled: true, storeErr: context.Canceled, serviceErr: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &memorySettings{values: make(map[string]string)}
			if tc.present {
				store.values["key"] = tc.raw
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tc.canceled {
				cancel()
			}

			got, err := store.Get(ctx, "key")
			if got != tc.raw || !errors.Is(err, tc.storeErr) {
				t.Fatalf("adapter: got %q, %v; want %q, %v", got, err, tc.raw, tc.storeErr)
			}

			registry := settings.NewRegistry()
			registry.Register(settings.Schema{Key: "key", Type: settings.String, Default: "fallback"})
			service := settings.NewService(registry, store)

			got, err = service.GetString(ctx, "key")
			if got != tc.serviceValue || !errors.Is(err, tc.serviceErr) {
				t.Fatalf("service: got %q, %v; want %q, %v", got, err, tc.serviceValue, tc.serviceErr)
			}
		})
	}
}
