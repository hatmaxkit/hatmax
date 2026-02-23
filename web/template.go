package web

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/hatmaxkit/hatmax/log"
)

type TemplateManager struct {
	fs        embed.FS
	templates *template.Template
	funcMap   template.FuncMap
	log       log.Logger
}

type TemplateOption func(*TemplateManager)

func WithFuncMap(fm template.FuncMap) TemplateOption {
	return func(m *TemplateManager) {
		m.funcMap = fm
	}
}

func NewTemplateManager(fs embed.FS, log log.Logger, opts ...TemplateOption) *TemplateManager {
	m := &TemplateManager{
		fs:  fs,
		log: log,
	}
	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *TemplateManager) Start(ctx context.Context) error {
	tmpl := template.New("")
	if m.funcMap != nil {
		tmpl = tmpl.Funcs(m.funcMap)
	}

	err := fs.WalkDir(m.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}

		content, err := m.fs.ReadFile(path)
		if err != nil {
			return err
		}

		normalizedPath := m.normalizePath(path)

		_, err = tmpl.New(normalizedPath).Parse(string(content))

		return err
	})
	if err != nil {
		m.log.Errorf("error parsing templates: %v", err)

		return err
	}

	m.templates = tmpl
	m.log.Info("templates loaded successfully")

	return nil
}

func (m *TemplateManager) Stop(ctx context.Context) error {
	return nil
}

func (m *TemplateManager) Render(w http.ResponseWriter, namespace, template string, data interface{}) {
	path := m.buildPath(namespace, template)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := m.templates.ExecuteTemplate(w, path, data)
	if err != nil {
		m.log.Errorf("error rendering template %s/%s: %v", namespace, template, err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
	}
}

func (m *TemplateManager) RenderPartial(w http.ResponseWriter, namespace, template string, data interface{}) {
	path := m.buildPath(namespace, template)

	err := m.templates.ExecuteTemplate(w, path, data)
	if err != nil {
		m.log.Errorf("error rendering partial %s/%s: %v", namespace, template, err)
		http.Error(w, "Partial rendering error", http.StatusInternalServerError)
	}
}

func (m *TemplateManager) buildPath(namespace, template string) string {
	return filepath.Join("assets", "templates", namespace, template+".html")
}

func (m *TemplateManager) normalizePath(path string) string {
	if idx := strings.Index(path, "assets"); idx >= 0 {
		return path[idx:]
	}

	return path
}
