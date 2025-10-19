package core

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"github.com/gertd/go-pluralize"
)

const templateBasePath = "assets/templates"

type TemplateManager struct {
	log       Logger
	assetsFS  embed.FS
	templates map[string]*template.Template
	mutex     sync.RWMutex
}

func NewTemplateManager(assetsFS embed.FS, log Logger) *TemplateManager {
	return &TemplateManager{
		log:       log,
		assetsFS:  assetsFS,
		templates: make(map[string]*template.Template),
	}
}

func (tm *TemplateManager) Start(ctx context.Context) error {
	tm.log.Info("Starting template manager...")

	if err := tm.parseTemplates(); err != nil {
		return fmt.Errorf("error parsing templates: %w", err)
	}

	tm.log.Info("Template manager started successfully", "count", len(tm.templates))
	return nil
}

func (tm *TemplateManager) parseTemplates() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	var allPaths []string

	sharedEntries, err := tm.assetsFS.ReadDir(templateBasePath + "/shared")
	if err != nil {
		return fmt.Errorf("error reading shared templates: %w", err)
	}

	for _, entry := range sharedEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		allPaths = append(allPaths, templateBasePath+"/shared/"+entry.Name())
	}

	rootEntries, err := tm.assetsFS.ReadDir(templateBasePath)
	if err != nil {
		return fmt.Errorf("error reading template base path: %w", err)
	}

	var handlerDirs []string
	for _, entry := range rootEntries {
		if entry.IsDir() && entry.Name() != "shared" {
			handlerDirs = append(handlerDirs, entry.Name())
		}
	}

	for _, handlerDir := range handlerDirs {
		entries, err := tm.assetsFS.ReadDir(templateBasePath + "/" + handlerDir)
		if err != nil {
			tm.log.Error("error reading handler templates", "path", handlerDir, "error", err)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}
			allPaths = append(allPaths, templateBasePath+"/"+handlerDir+"/"+entry.Name())
		}
	}

	for _, handlerDir := range handlerDirs {
		entries, err := tm.assetsFS.ReadDir(templateBasePath + "/" + handlerDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}

			name := entry.Name()
			tmpl := template.New(name)
			tmpl, err = tmpl.ParseFS(tm.assetsFS, allPaths...)
			if err != nil {
				return fmt.Errorf("error parsing template %s: %w", name, err)
			}

			tm.templates[name] = tmpl
			tm.log.Info("Loaded template", "name", name)
		}
	}

	for _, entry := range sharedEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		name := entry.Name()
		if name == "base.html" {
			continue
		}

		tmpl := template.New(name)
		tmpl, err = tmpl.ParseFS(tm.assetsFS, allPaths...)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", name, err)
		}

		tm.templates[name] = tmpl
		tm.log.Info("Loaded template", "name", name)
	}

	return nil
}

func (tm *TemplateManager) Get(name string) (*template.Template, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	tmpl, ok := tm.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %s not found", name)
	}

	return tmpl, nil
}

func (tm *TemplateManager) GetByPath(handler, action string) (*template.Template, error) {
	pluralizer := pluralize.NewClient()

	var name string
	switch action {
	case "list":
		name = pluralizer.Plural(handler) + ".html"
	case "new", "edit", "show":
		name = action + "-" + handler + ".html"
	default:
		name = action + "-" + handler + ".html"
	}

	return tm.Get(name)
}

func (tm *TemplateManager) Reload() error {
	tm.log.Info("Reloading templates...")
	return tm.parseTemplates()
}
