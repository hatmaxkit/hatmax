package web

import (
	"context"
	"embed"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hatmaxkit/hatmax/log"
)

//go:embed testdata/assets/templates/test/page.html testdata/assets/templates/test/item.html
var testFS embed.FS

//go:embed testdata/assets/templates/test/invalid.html
var invalidFS embed.FS

func TestNewTemplateManager(t *testing.T) {
	logger := log.NewTestLogger("error")
	mgr := NewTemplateManager(testFS, logger)

	if mgr == nil {
		t.Fatal("NewTemplateManager() returned nil")
	}

	if mgr.log == nil {
		t.Error("NewTemplateManager() logger is nil")
	}

	if mgr.templates != nil {
		t.Error("NewTemplateManager() templates should be nil before Start()")
	}
}

func TestTemplateManagerStart(t *testing.T) {
	tests := []struct {
		name    string
		fs      embed.FS
		wantErr bool
	}{
		{
			name:    "valid templates",
			fs:      testFS,
			wantErr: false,
		},
		{
			name:    "invalid template syntax",
			fs:      invalidFS,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewTestLogger("error")
			mgr := NewTemplateManager(tt.fs, logger)

			err := mgr.Start(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && mgr.templates == nil {
				t.Error("Start() templates is nil after successful start")
			}
		})
	}
}

func TestTemplateManagerStop(t *testing.T) {
	logger := log.NewTestLogger("error")
	mgr := NewTemplateManager(testFS, logger)

	err := mgr.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}
}

func TestTemplateManagerRender(t *testing.T) {
	logger := log.NewTestLogger("error")
	mgr := NewTemplateManager(testFS, logger)
	mgr.Start(context.Background())

	tests := []struct {
		name        string
		namespace   string
		template    string
		data        interface{}
		wantStatus  int
		wantContain string
	}{
		{
			name:        "valid template",
			namespace:   "test",
			template:    "page",
			data:        map[string]string{"Title": "Test Page"},
			wantStatus:  200,
			wantContain: "Test Page",
		},
		{
			name:        "template not found",
			namespace:   "missing",
			template:    "page",
			data:        nil,
			wantStatus:  500,
			wantContain: "Template rendering error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			mgr.Render(w, tt.namespace, tt.template, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("Render() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if !strings.Contains(w.Body.String(), tt.wantContain) {
				t.Errorf("Render() body = %q, want to contain %q", w.Body.String(), tt.wantContain)
			}

			if tt.wantStatus == 200 {
				contentType := w.Header().Get("Content-Type")
				if contentType != "text/html; charset=utf-8" {
					t.Errorf("Render() Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
				}
			}
		})
	}
}

func TestTemplateManagerRenderPartial(t *testing.T) {
	logger := log.NewTestLogger("error")
	mgr := NewTemplateManager(testFS, logger)
	mgr.Start(context.Background())

	tests := []struct {
		name        string
		namespace   string
		template    string
		data        interface{}
		wantStatus  int
		wantContain string
	}{
		{
			name:        "valid partial",
			namespace:   "test",
			template:    "item",
			data:        map[string]string{"Name": "Item"},
			wantStatus:  200,
			wantContain: "Item",
		},
		{
			name:        "partial not found",
			namespace:   "missing",
			template:    "item",
			data:        nil,
			wantStatus:  500,
			wantContain: "Partial rendering error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			mgr.RenderPartial(w, tt.namespace, tt.template, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("RenderPartial() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if !strings.Contains(w.Body.String(), tt.wantContain) {
				t.Errorf("RenderPartial() body = %q, want to contain %q", w.Body.String(), tt.wantContain)
			}
		})
	}
}
