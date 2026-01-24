package middleware

import "net/http"

// StaticCache adds aggressive cache headers for static assets.
// Safe for content-addressed files where URLs change when content changes.
func StaticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
