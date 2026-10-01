// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/log"
)

// A start-only component must not borrow the failed component's stop function.
func TestStartOnlyRollback(t *testing.T) {
	want := errors.New("startup failed")
	started := &fakeStartable{}
	failed := &fakeComponent{fakeStartable: fakeStartable{err: want}}
	router := chi.NewRouter()
	starts, stops, registrars := Setup(context.Background(), router, started, failed)

	err := Start(context.Background(), log.NewTestLogger("error"), starts, stops, registrars, router)
	if err != want {
		t.Fatalf("Start() = %v, want original error %v", err, want)
	}

	if failed.stopped.Load() {
		t.Fatal("failed component was stopped instead of the start-only component")
	}

	if failed.registered {
		t.Fatal("routes were registered after startup failure")
	}
}

type lifecycleProbe struct {
	t        *testing.T
	name     string
	events   *[]string
	startCtx context.Context
	startErr error
	stopErr  error
	cancel   context.CancelFunc
}

func (probe *lifecycleProbe) Start(ctx context.Context) error {
	probe.t.Helper()

	if ctx != probe.startCtx {
		probe.t.Fatal("start did not receive the caller context")
	}

	*probe.events = append(*probe.events, "start:"+probe.name)
	if probe.startErr != nil {
		probe.cancel()
	}

	return probe.startErr
}

func (probe *lifecycleProbe) Stop(ctx context.Context) error {
	probe.t.Helper()

	if ctx != context.Background() {
		probe.t.Fatal("stop did not receive the independent background context")
	}

	*probe.events = append(*probe.events, "stop:"+probe.name)

	return probe.stopErr
}

func (probe *lifecycleProbe) RegisterRoutes(_ chi.Router) {
	*probe.events = append(*probe.events, "routes:"+probe.name)
}

type rollbackLogger struct {
	log.Logger
	errors []string
}

func (logger *rollbackLogger) Errorf(format string, args ...any) {
	logger.errors = append(logger.errors, fmt.Sprintf(format, args...))
}

// Mixed capabilities must retain identity at every failure index and gate routes.
func TestMixedRollback(t *testing.T) {
	cases := []struct {
		name       string
		failure    int
		stopFails  bool
		wantEvents []string
	}{
		{name: "first", failure: 0, wantEvents: []string{"start:start1"}},
		{name: "second", failure: 1, wantEvents: []string{"start:start1", "start:pair1"}},
		{name: "third", failure: 2, wantEvents: []string{"start:start1", "start:pair1", "start:pair2", "stop:pair1"}},
		{name: "fourth", failure: 3, wantEvents: []string{"start:start1", "start:pair1", "start:pair2", "start:start2", "stop:pair2", "stop:pair1"}},
		{name: "last", failure: 4, wantEvents: []string{"start:start1", "start:pair1", "start:pair2", "start:start2", "start:pair3", "stop:pair2", "stop:pair1"}},
		{name: "stop_error", failure: 4, stopFails: true, wantEvents: []string{"start:start1", "start:pair1", "start:pair2", "start:start2", "start:pair3", "stop:pair2", "stop:pair1"}},
		{name: "success", failure: -1, wantEvents: []string{"start:start1", "start:pair1", "start:pair2", "start:start2", "start:pair3", "routes:pair1", "routes:route1", "routes:pair2", "routes:pair3", "routes:route2"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var events []string

			probes := make(map[string]*lifecycleProbe)
			for _, name := range []string{"stop1", "start1", "pair1", "route1", "pair2", "start2", "pair3", "stop2", "route2"} {
				probes[name] = &lifecycleProbe{t: t, name: name, events: &events, startCtx: ctx, cancel: cancel}
			}

			var wantErr error
			if test.failure >= 0 {
				wantErr = errors.New("original start error")
				name := []string{"start1", "pair1", "pair2", "start2", "pair3"}[test.failure]
				probes[name].startErr = wantErr
			}

			if test.stopFails {
				probes["pair2"].stopErr = errors.New("rollback stop error")
			}

			components := []any{
				struct{ Stoppable }{probes["stop1"]},
				struct{ Startable }{probes["start1"]},
				probes["pair1"],
				struct{ RouteRegistrar }{probes["route1"]},
				probes["pair2"],
				struct{ Startable }{probes["start2"]},
				probes["pair3"],
				struct{ Stoppable }{probes["stop2"]},
				struct{ RouteRegistrar }{probes["route2"]},
			}
			router := chi.NewRouter()

			starts, stops, registrars := Setup(ctx, router, components...)
			if len(starts) != 5 || len(stops) != 5 || len(registrars) != 5 || len(events) != 0 {
				t.Fatalf("Setup() sizes = %d/%d/%d, events = %v", len(starts), len(stops), len(registrars), events)
			}

			logger := &rollbackLogger{Logger: log.NewNoopLogger()}

			err := Start(ctx, logger, starts, stops, registrars, router)
			if err != wantErr {
				t.Fatalf("Start() = %v, want original error %v", err, wantErr)
			}

			if !slices.Equal(events, test.wantEvents) {
				t.Fatalf("events = %v, want %v", events, test.wantEvents)
			}

			if test.stopFails {
				wantLog := "error stopping component #2 during rollback: rollback stop error"
				if !slices.Contains(logger.errors, wantLog) {
					t.Errorf("logged errors = %v, want %q", logger.errors, wantLog)
				}
			}
		})
	}
}

// No Stop capability is valid even when several completed starts precede failure.
func TestRollbackWithoutStops(t *testing.T) {
	want := errors.New("start failed")
	router := chi.NewRouter()
	starts, stops, registrars := Setup(context.Background(), router,
		&fakeStartable{}, &fakeStartable{}, &fakeStartable{err: want},
	)

	err := Start(context.Background(), log.NewNoopLogger(), starts, stops, registrars, router)
	if err != want {
		t.Fatalf("Start() = %v, want original error %v", err, want)
	}
}

// Normal shutdown must still include stop-only components in component order.
func TestMixedShutdown(t *testing.T) {
	var events []string

	first := &lifecycleProbe{t: t, name: "first", events: &events}
	paired := &lifecycleProbe{t: t, name: "paired", events: &events}
	last := &lifecycleProbe{t: t, name: "last", events: &events}
	router := chi.NewRouter()
	_, stops, _ := Setup(context.Background(), router,
		struct{ Stoppable }{first}, &fakeStartable{}, paired,
		&fakeRouteRegistrar{}, struct{ Stoppable }{last},
	)

	Shutdown(&http.Server{}, log.NewNoopLogger(), stops)

	if want := []string{"stop:last", "stop:paired", "stop:first"}; !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}
