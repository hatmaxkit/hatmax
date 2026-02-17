package kit

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Assets manages static assets with overlay support.
type Assets struct {
	embeds  embed.FS
	overlay fs.FS
	prefix  string
}

// NewAssets creates a new asset manager.
func NewAssets(embeds embed.FS) *Assets {
	return &Assets{
		embeds: embeds,
		prefix: "/static",
	}
}

// WithOverlay sets a filesystem overlay for asset customization.
func (a *Assets) WithOverlay(overlay fs.FS) *Assets {
	a.overlay = overlay
	return a
}

// WithPrefix sets the URL prefix.
func (a *Assets) WithPrefix(prefix string) *Assets {
	a.prefix = prefix
	return a
}

// Prefix returns the URL prefix.
func (a *Assets) Prefix() string {
	return a.prefix
}

// URL returns the full URL for an asset.
func (a *Assets) URL(assetPath string) string {
	if !strings.HasPrefix(assetPath, "/") {
		assetPath = "/" + assetPath
	}
	return a.prefix + assetPath
}

// Open opens a file, checking overlay first.
func (a *Assets) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(name, "/")

	if a.overlay != nil {
		if f, err := a.overlay.Open(name); err == nil {
			return f, nil
		}
	}

	return a.embeds.Open(name)
}

// Read reads a file completely.
func (a *Assets) Read(name string) ([]byte, error) {
	f, err := a.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

// Handler returns an http.Handler for serving assets.
func (a *Assets) Handler() http.Handler {
	return http.StripPrefix(a.prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			http.NotFound(w, r)
			return
		}

		f, err := a.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if stat.IsDir() {
			http.NotFound(w, r)
			return
		}

		content, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		contentType := getContentType(name)
		w.Header().Set("Content-Type", contentType)
		w.Write(content)
	}))
}

func getContentType(name string) string {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}

// Assets returns an asset manager for the kit.
func (k *Kit) Assets() *Assets {
	assets := NewAssets(k.embeds)
	if k.overlay != nil {
		assets.WithOverlay(k.overlay)
	}
	return assets
}
