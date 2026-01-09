package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/log"
)

type fakeRouteRegistrar struct {
	registered bool
}

func (f *fakeRouteRegistrar) RegisterRoutes(r chi.Router) {
	f.registered = true
}

type fakeStartable struct {
	started bool
	err     error
}

func (f *fakeStartable) Start(ctx context.Context) error {
	if f.err != nil {
		return f.err
	}
	f.started = true
	return nil
}

type fakeStoppable struct {
	stopped atomic.Bool
	err     error
}

func (f *fakeStoppable) Stop(ctx context.Context) error {
	if f.err != nil {
		return f.err
	}
	f.stopped.Store(true)
	return nil
}

type fakeComponent struct {
	fakeRouteRegistrar
	fakeStartable
	fakeStoppable
}

func TestSetupWithNoComponents(t *testing.T) {
	r := chi.NewRouter()
	starts, stops, registrars := Setup(context.Background(), r)

	if len(starts) != 0 {
		t.Errorf("expected 0 starts, got %d", len(starts))
	}
	if len(stops) != 0 {
		t.Errorf("expected 0 stops, got %d", len(stops))
	}
	if len(registrars) != 0 {
		t.Errorf("expected 0 registrars, got %d", len(registrars))
	}
}

func TestSetupWithRouteRegistrar(t *testing.T) {
	r := chi.NewRouter()
	comp := &fakeRouteRegistrar{}

	starts, stops, registrars := Setup(context.Background(), r, comp)

	if comp.registered {
		t.Error("expected RegisterRoutes NOT to be called during Setup")
	}
	if len(starts) != 0 {
		t.Errorf("expected 0 starts, got %d", len(starts))
	}
	if len(stops) != 0 {
		t.Errorf("expected 0 stops, got %d", len(stops))
	}
	if len(registrars) != 1 {
		t.Errorf("expected 1 registrar, got %d", len(registrars))
	}
}

func TestSetupWithStartable(t *testing.T) {
	r := chi.NewRouter()
	comp := &fakeStartable{}

	starts, stops, _ := Setup(context.Background(), r, comp)

	if len(starts) != 1 {
		t.Errorf("expected 1 start, got %d", len(starts))
	}
	if len(stops) != 0 {
		t.Errorf("expected 0 stops, got %d", len(stops))
	}
}

func TestSetupWithStoppable(t *testing.T) {
	r := chi.NewRouter()
	comp := &fakeStoppable{}

	starts, stops, _ := Setup(context.Background(), r, comp)

	if len(starts) != 0 {
		t.Errorf("expected 0 starts, got %d", len(starts))
	}
	if len(stops) != 1 {
		t.Errorf("expected 1 stop, got %d", len(stops))
	}
}

func TestSetupWithFullComponent(t *testing.T) {
	r := chi.NewRouter()
	comp := &fakeComponent{}

	starts, stops, registrars := Setup(context.Background(), r, comp)

	if comp.registered {
		t.Error("expected RegisterRoutes NOT to be called during Setup")
	}
	if len(starts) != 1 {
		t.Errorf("expected 1 start, got %d", len(starts))
	}
	if len(stops) != 1 {
		t.Errorf("expected 1 stop, got %d", len(stops))
	}
	if len(registrars) != 1 {
		t.Errorf("expected 1 registrar, got %d", len(registrars))
	}
}

func TestSetupWithMultipleComponents(t *testing.T) {
	r := chi.NewRouter()
	comp1 := &fakeComponent{}
	comp2 := &fakeComponent{}
	comp3 := &fakeRouteRegistrar{}

	starts, stops, registrars := Setup(context.Background(), r, comp1, comp2, comp3)

	if comp1.registered || comp2.registered || comp3.registered {
		t.Error("expected RegisterRoutes NOT to be called during Setup")
	}
	if len(starts) != 2 {
		t.Errorf("expected 2 starts, got %d", len(starts))
	}
	if len(stops) != 2 {
		t.Errorf("expected 2 stops, got %d", len(stops))
	}
	if len(registrars) != 3 {
		t.Errorf("expected 3 registrars, got %d", len(registrars))
	}
}

func TestStartSuccess(t *testing.T) {
	comp1 := &fakeComponent{}
	comp2 := &fakeComponent{}

	starts := []func(context.Context) error{
		comp1.Start,
		comp2.Start,
	}
	stops := []func(context.Context) error{
		comp1.Stop,
		comp2.Stop,
	}
	registrars := []RouteRegistrar{comp1, comp2}
	r := chi.NewRouter()

	logger := log.NewTestLogger("error")
	err := Start(context.Background(), logger, starts, stops, registrars, r)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !comp1.started {
		t.Error("expected comp1 to be started")
	}
	if !comp2.started {
		t.Error("expected comp2 to be started")
	}
	if !comp1.registered {
		t.Error("expected comp1 routes to be registered after Start")
	}
	if !comp2.registered {
		t.Error("expected comp2 routes to be registered after Start")
	}
}

func TestStartWithFirstComponentFailure(t *testing.T) {
	testErr := errors.New("start error")
	comp1 := &fakeComponent{fakeStartable: fakeStartable{err: testErr}}
	comp2 := &fakeComponent{}

	starts := []func(context.Context) error{
		comp1.Start,
		comp2.Start,
	}
	stops := []func(context.Context) error{
		comp1.Stop,
		comp2.Stop,
	}

	r := chi.NewRouter()
	logger := log.NewTestLogger("error")
	err := Start(context.Background(), logger, starts, stops, nil, r)

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if comp1.started {
		t.Error("expected comp1 not to be started")
	}
	if comp2.started {
		t.Error("expected comp2 not to be started")
	}
	if comp1.stopped.Load() {
		t.Error("expected comp1 not to be stopped (never started)")
	}
	if comp2.stopped.Load() {
		t.Error("expected comp2 not to be stopped (never started)")
	}
}

func TestStartWithSecondComponentFailure(t *testing.T) {
	testErr := errors.New("start error")
	comp1 := &fakeComponent{}
	comp2 := &fakeComponent{fakeStartable: fakeStartable{err: testErr}}

	starts := []func(context.Context) error{
		comp1.Start,
		comp2.Start,
	}
	stops := []func(context.Context) error{
		comp1.Stop,
		comp2.Stop,
	}

	r := chi.NewRouter()
	logger := log.NewTestLogger("error")
	err := Start(context.Background(), logger, starts, stops, nil, r)

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if !comp1.started {
		t.Error("expected comp1 to be started")
	}
	if comp2.started {
		t.Error("expected comp2 not to be started")
	}
	if !comp1.stopped.Load() {
		t.Error("expected comp1 to be stopped (rollback)")
	}
	if comp2.stopped.Load() {
		t.Error("expected comp2 not to be stopped (never started)")
	}
}

func TestStartWithRollbackFailure(t *testing.T) {
	testErr := errors.New("start error")
	stopErr := errors.New("stop error")
	comp1 := &fakeComponent{fakeStoppable: fakeStoppable{err: stopErr}}
	comp2 := &fakeComponent{fakeStartable: fakeStartable{err: testErr}}

	starts := []func(context.Context) error{
		comp1.Start,
		comp2.Start,
	}
	stops := []func(context.Context) error{
		comp1.Stop,
		comp2.Stop,
	}

	r := chi.NewRouter()
	logger := log.NewTestLogger("error")
	err := Start(context.Background(), logger, starts, stops, nil, r)

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if !comp1.started {
		t.Error("expected comp1 to be started")
	}
}

func TestShutdown(t *testing.T) {
	comp1 := &fakeComponent{}
	comp2 := &fakeComponent{}

	stops := []func(context.Context) error{
		comp1.Stop,
		comp2.Stop,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	logger := log.NewTestLogger("error")

	go func() {
		time.Sleep(100 * time.Millisecond)
		Shutdown(srv.Config, logger, stops)
	}()

	time.Sleep(200 * time.Millisecond)

	if !comp1.stopped.Load() || !comp2.stopped.Load() {
		t.Error("expected both components to be stopped")
	}
}

func TestShutdownStopsInReverseOrder(t *testing.T) {
	var stopOrder []int

	comp1 := &fakeStoppable{}
	comp1Stop := func(ctx context.Context) error {
		stopOrder = append(stopOrder, 1)
		return comp1.Stop(ctx)
	}

	comp2 := &fakeStoppable{}
	comp2Stop := func(ctx context.Context) error {
		stopOrder = append(stopOrder, 2)
		return comp2.Stop(ctx)
	}

	comp3 := &fakeStoppable{}
	comp3Stop := func(ctx context.Context) error {
		stopOrder = append(stopOrder, 3)
		return comp3.Stop(ctx)
	}

	stops := []func(context.Context) error{
		comp1Stop,
		comp2Stop,
		comp3Stop,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	logger := log.NewTestLogger("error")
	Shutdown(srv.Config, logger, stops)

	expectedOrder := []int{3, 2, 1}
	if len(stopOrder) != len(expectedOrder) {
		t.Fatalf("expected %d stops, got %d", len(expectedOrder), len(stopOrder))
	}
	for i, expected := range expectedOrder {
		if stopOrder[i] != expected {
			t.Errorf("stop order[%d] = %d, want %d", i, stopOrder[i], expected)
		}
	}
}

func TestShutdownWithStopError(t *testing.T) {
	stopErr := errors.New("stop error")
	comp1 := &fakeComponent{fakeStoppable: fakeStoppable{err: stopErr}}
	comp2 := &fakeComponent{}

	stops := []func(context.Context) error{
		comp1.Stop,
		comp2.Stop,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	logger := log.NewTestLogger("error")
	Shutdown(srv.Config, logger, stops)

	if !comp2.stopped.Load() {
		t.Error("expected comp2 to be stopped despite comp1 error")
	}
}
