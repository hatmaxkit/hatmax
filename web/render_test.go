package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatmax.adrianpk.com/log"
)

func TestRenderTemplate(t *testing.T) {
	logger := log.NewTestLogger("error")

	tests := []struct {
		name           string
		templateStr    string
		templateName   string
		data           interface{}
		wantStatus     int
		wantBody       string
		wantHeaderType string
	}{
		{
			name:           "success",
			templateStr:    `{{define "test"}}Hello {{.Name}}{{end}}`,
			templateName:   "test",
			data:           map[string]string{"Name": "World"},
			wantStatus:     http.StatusOK,
			wantBody:       "Hello World",
			wantHeaderType: "text/html; charset=utf-8",
		},
		{
			name:           "template error",
			templateStr:    `{{define "test"}}Valid{{end}}`,
			templateName:   "nonexistent",
			data:           nil,
			wantStatus:     http.StatusInternalServerError,
			wantBody:       "Template rendering error",
			wantHeaderType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := template.Must(template.New("").Parse(tt.templateStr))
			w := httptest.NewRecorder()

			RenderTemplate(w, tmpl, tt.templateName, tt.data, logger)

			if w.Code != tt.wantStatus {
				t.Errorf("RenderTemplate() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("RenderTemplate() body = %v, want to contain %v", w.Body.String(), tt.wantBody)
			}

			if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, tt.wantHeaderType) {
				t.Errorf("RenderTemplate() Content-Type = %v, want %v", contentType, tt.wantHeaderType)
			}
		})
	}
}

func TestRenderPartial(t *testing.T) {
	logger := log.NewTestLogger("error")

	tests := []struct {
		name         string
		templateStr  string
		templateName string
		data         interface{}
		wantStatus   int
		wantBody     string
	}{
		{
			name:         "success",
			templateStr:  `{{define "item"}}<div>{{.ID}}</div>{{end}}`,
			templateName: "item",
			data:         map[string]string{"ID": "123"},
			wantStatus:   http.StatusOK,
			wantBody:     "<div>123</div>",
		},
		{
			name:         "template error",
			templateStr:  `{{define "item"}}Valid{{end}}`,
			templateName: "missing",
			data:         nil,
			wantStatus:   http.StatusInternalServerError,
			wantBody:     "Partial rendering error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := template.Must(template.New("").Parse(tt.templateStr))
			w := httptest.NewRecorder()

			RenderPartial(w, tmpl, tt.templateName, tt.data, logger)

			if w.Code != tt.wantStatus {
				t.Errorf("RenderPartial() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("RenderPartial() body = %v, want to contain %v", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRedirectOrHXRedirect(t *testing.T) {
	tests := []struct {
		name           string
		hxRequest      bool
		url            string
		wantStatus     int
		wantLocation   string
		wantHXRedirect string
	}{
		{
			name:         "normal redirect",
			hxRequest:    false,
			url:          "/target",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/target",
		},
		{
			name:           "htmx redirect",
			hxRequest:      true,
			url:            "/htmx-target",
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/htmx-target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			if tt.hxRequest {
				r.Header.Set("HX-Request", "true")
			}

			RedirectOrHXRedirect(w, r, tt.url)

			if w.Code != tt.wantStatus {
				t.Errorf("RedirectOrHXRedirect() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if tt.wantLocation != "" {
				if location := w.Header().Get("Location"); location != tt.wantLocation {
					t.Errorf("RedirectOrHXRedirect() Location = %v, want %v", location, tt.wantLocation)
				}
			}

			if tt.wantHXRedirect != "" {
				if hxRedirect := w.Header().Get("HX-Redirect"); hxRedirect != tt.wantHXRedirect {
					t.Errorf("RedirectOrHXRedirect() HX-Redirect = %v, want %v", hxRedirect, tt.wantHXRedirect)
				}
			}
		})
	}
}
