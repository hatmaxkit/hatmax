// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/middleware"
	"hatmax.adrianpk.com/model"
	"hatmax.adrianpk.com/pubsub"
	pubsubpostgres "hatmax.adrianpk.com/pubsub/postgres"
	"hatmax.adrianpk.com/settings"
	"hatmax.adrianpk.com/slug"
	"hatmax.adrianpk.com/validation"
	"hatmax.adrianpk.com/web"
)

//go:embed assets
var assetsFS embed.FS

type memorySettings struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySettings) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok := m.values[key]
	if !ok {
		return "", errors.New("setting not found")
	}

	return value, nil
}

func (m *memorySettings) Set(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.values == nil {
		m.values = make(map[string]string)
	}

	m.values[key] = value

	return nil
}

func (m *memorySettings) All(context.Context) ([]settings.Value, error) {
	return nil, nil
}

func (m *memorySettings) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.values, key)

	return nil
}

type note struct {
	Title string
	Slug  string
}

type noteStore struct {
	database *db.Database
}

func (s *noteStore) Start(ctx context.Context) error {
	_, err := s.database.GetDB().ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS guide_notes (
			id text PRIMARY KEY,
			title text NOT NULL,
			slug text NOT NULL,
			created_at timestamptz NOT NULL
		)
	`)

	return err
}

func (s *noteStore) Stop(context.Context) error { return nil }

func (s *noteStore) insert(ctx context.Context, title string) error {
	id := model.NewID()

	parsed, err := model.ParseID(id)
	if err != nil {
		return err
	}

	_, err = s.database.GetDB().ExecContext(ctx, `
		INSERT INTO guide_notes (id, title, slug, created_at)
		VALUES ($1, $2, $3, $4)
	`, id, title, slug.Generate(title, parsed), model.Now())

	return err
}

func (s *noteStore) list(ctx context.Context) ([]note, error) {
	rows, err := s.database.GetDB().QueryContext(ctx, `
		SELECT title, slug FROM guide_notes ORDER BY created_at, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []note

	for rows.Next() {
		var item note

		err = rows.Scan(&item.Title, &item.Slug)
		if err != nil {
			return nil, err
		}

		notes = append(notes, item)
	}

	return notes, rows.Err()
}

type eventListener struct {
	broker *pubsubpostgres.Broker
	mu     sync.Mutex
	seen   []string
}

func (l *eventListener) Start(ctx context.Context) error {
	return l.broker.Subscribe(ctx, "guide.notes", l.handle, pubsub.SubscribeOptions{
		SubscriberID: "guide-notes",
	})
}

func (l *eventListener) Stop(context.Context) error { return nil }

func (l *eventListener) handle(_ context.Context, envelope pubsub.Envelope) error {
	title, ok := envelope.Payload.(string)
	if !ok {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.seen = append(l.seen, title)

	return nil
}

func (l *eventListener) values() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]string(nil), l.seen...)
}

type pages struct {
	templates *web.TemplateManager
	settings  *settings.Service
	notes     *noteStore
	broker    *pubsubpostgres.Broker
	listener  *eventListener
}

type pageView struct {
	Greeting        string
	DatabaseEnabled bool
	Notes           []note
}

func (p *pages) RegisterRoutes(router chi.Router) {
	router.Get("/", p.home)
	router.Post("/name", p.checkName)
	router.Get("/greeting", p.greeting)
	router.Post("/greeting", p.setGreeting)
	router.Get("/effects", p.effects)

	if p.notes != nil {
		router.Post("/notes", p.saveNote)
	}
}

func (p *pages) home(w http.ResponseWriter, r *http.Request) {
	greeting, err := p.settings.GetString(r.Context(), "guide.greeting")
	if err != nil {
		http.Error(w, "cannot read greeting", http.StatusInternalServerError)

		return
	}

	view := pageView{Greeting: greeting, DatabaseEnabled: p.notes != nil}
	if p.notes != nil {
		view.Notes, err = p.notes.list(r.Context())
		if err != nil {
			http.Error(w, "cannot list notes", http.StatusInternalServerError)

			return
		}
	}

	p.templates.Render(w, "home", "page", view)
}

func (p *pages) checkName(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	errs := validation.Field("name", name).Required().MinLength(2).Errors()

	message := "Accepted " + name
	if errs.HasErrors() {
		message = errs.Error()
	}

	p.templates.RenderPartial(w, "home", "status", map[string]string{"Message": message})
}

func (p *pages) greeting(w http.ResponseWriter, r *http.Request) {
	value, err := p.settings.GetString(r.Context(), "guide.greeting")
	if err != nil {
		http.Error(w, "cannot read greeting", http.StatusInternalServerError)

		return
	}

	fmt.Fprintln(w, value)
}

func (p *pages) setGreeting(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	err = p.settings.Set(r.Context(), "guide.greeting", r.FormValue("greeting"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *pages) saveNote(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)

		return
	}

	err = p.notes.insert(r.Context(), title)
	if err != nil {
		http.Error(w, "cannot save note", http.StatusInternalServerError)

		return
	}

	envelope := pubsub.NewEnvelope("guide.notes", title)

	err = p.broker.Publish(r.Context(), "guide.notes", envelope)
	if err != nil {
		http.Error(w, "cannot publish note", http.StatusInternalServerError)

		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *pages) effects(w http.ResponseWriter, _ *http.Request) {
	if p.listener == nil {
		return
	}

	for _, title := range p.listener.values() {
		fmt.Fprintln(w, title)
	}
}

func main() {
	cfg, err := config.Load("config.yaml", "GUIDE_", os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load config: %v\n", err)
		os.Exit(1)
	}

	err = cfg.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot validate config: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewLogger(cfg)
	router := app.NewRouter(
		logger,
		app.WithMiddleware(middleware.RequireSameOrigin),
		app.WithPing(),
	)
	templates := web.NewTemplateManager(assetsFS, logger)

	registry := settings.NewRegistry()
	registry.Register(settings.Schema{
		Key:     "guide.greeting",
		Type:    settings.String,
		Default: "Hello",
	})
	settingService := settings.NewService(registry, &memorySettings{})

	pageHandlers := &pages{templates: templates, settings: settingService}
	components := []any{templates, pageHandlers}

	if strings.EqualFold(os.Getenv("GUIDE_DATABASE_ENABLED"), "true") {
		database := db.New(assetsFS, db.Postgres, cfg, logger)
		notes := &noteStore{database: database}
		broker := pubsubpostgres.New(database, cfg, logger)
		listener := &eventListener{broker: broker}
		pageHandlers.notes = notes
		pageHandlers.broker = broker
		pageHandlers.listener = listener
		components = []any{database, notes, broker, listener, templates, pageHandlers}
	}

	ctx := context.Background()

	starts, stops, registrars := app.Setup(ctx, router, components...)

	err = app.Start(ctx, logger, starts, stops, registrars, router)
	if err != nil {
		logger.Errorf("cannot start: %v", err)
		os.Exit(1)
	}

	err = app.Serve(router, cfg.Server.Port)
	if err != nil {
		logger.Errorf("cannot serve: %v", err)
		os.Exit(1)
	}
}
