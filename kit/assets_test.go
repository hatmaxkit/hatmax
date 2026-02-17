package kit

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

//go:embed testdata/*
var testEmbeds embed.FS

func TestAssetsURL(t *testing.T) {
	assets := NewAssets(testEmbeds)

	tests := []struct {
		path string
		want string
	}{
		{"style.css", "/static/style.css"},
		{"/style.css", "/static/style.css"},
		{"js/app.js", "/static/js/app.js"},
	}

	for _, tt := range tests {
		got := assets.URL(tt.path)
		if got != tt.want {
			t.Errorf("URL(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestAssetsPrefix(t *testing.T) {
	assets := NewAssets(testEmbeds).WithPrefix("/assets")

	if assets.Prefix() != "/assets" {
		t.Errorf("Prefix() = %q, want %q", assets.Prefix(), "/assets")
	}

	url := assets.URL("style.css")
	if url != "/assets/style.css" {
		t.Errorf("URL() = %q, want %q", url, "/assets/style.css")
	}
}

func TestAssetsOverlay(t *testing.T) {
	overlay := fstest.MapFS{
		"custom.css": &fstest.MapFile{Data: []byte("/* custom */")},
	}

	assets := NewAssets(testEmbeds).WithOverlay(overlay)

	content, err := assets.Read("custom.css")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(content) != "/* custom */" {
		t.Errorf("Read() = %q, want %q", string(content), "/* custom */")
	}
}

func TestAssetsOpenNotFound(t *testing.T) {
	assets := NewAssets(testEmbeds)

	_, err := assets.Open("nonexistent.css")
	if err == nil {
		t.Error("Open() expected error for nonexistent file")
	}
}

func TestAssetsHandler(t *testing.T) {
	overlay := fstest.MapFS{
		"test.css": &fstest.MapFile{Data: []byte("body { color: red; }")},
	}

	assets := NewAssets(testEmbeds).WithOverlay(overlay)
	handler := assets.Handler()

	tests := []struct {
		path       string
		wantStatus int
		wantType   string
	}{
		{"/static/test.css", http.StatusOK, "text/css; charset=utf-8"},
		{"/static/notfound.css", http.StatusNotFound, ""},
		{"/static/", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != tt.wantStatus {
			t.Errorf("Handler(%q) status = %d, want %d", tt.path, w.Code, tt.wantStatus)
		}

		if tt.wantType != "" && w.Header().Get("Content-Type") != tt.wantType {
			t.Errorf("Handler(%q) content-type = %q, want %q",
				tt.path, w.Header().Get("Content-Type"), tt.wantType)
		}
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"style.css", "text/css; charset=utf-8"},
		{"app.js", "application/javascript; charset=utf-8"},
		{"data.json", "application/json; charset=utf-8"},
		{"page.html", "text/html; charset=utf-8"},
		{"icon.svg", "image/svg+xml"},
		{"photo.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"anim.gif", "image/gif"},
		{"favicon.ico", "image/x-icon"},
		{"font.woff", "font/woff"},
		{"font.woff2", "font/woff2"},
		{"font.ttf", "font/ttf"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		got := getContentType(tt.name)
		if got != tt.want {
			t.Errorf("getContentType(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAssetsHandlerDirectory(t *testing.T) {
	overlay := fstest.MapFS{
		"subdir/file.txt": &fstest.MapFile{Data: []byte("test")},
	}

	assets := NewAssets(testEmbeds).WithOverlay(overlay)
	handler := assets.Handler()

	req := httptest.NewRequest(http.MethodGet, "/subdir", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Handler for directory status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

type errorFS struct{}

func (errorFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func TestAssetsOverlayFallback(t *testing.T) {
	assets := NewAssets(testEmbeds).WithOverlay(errorFS{})

	_, err := assets.Open("testdata/sample.txt")
	if err != nil {
		t.Errorf("Open() should fall back to embeds, got error: %v", err)
	}
}
