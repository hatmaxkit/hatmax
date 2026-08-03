package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// RequireSameOrigin rejects state-changing browser requests without a matching
// Origin or Referer header.
func RequireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if safeMethod(request.Method) {
			next.ServeHTTP(response, request)

			return
		}

		if !sameOriginRequest(request) {
			http.Error(response, "Forbidden", http.StatusForbidden)

			return
		}

		next.ServeHTTP(response, request)
	})
}

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func sameOriginRequest(request *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(request.Header.Get("Sec-Fetch-Site")), "cross-site") {
		return false
	}

	source := strings.TrimSpace(request.Header.Get("Origin"))
	if source == "" {
		source = strings.TrimSpace(request.Header.Get("Referer"))
	}

	sourceURL, err := url.Parse(source)
	if err != nil || sourceURL.Scheme == "" || sourceURL.Host == "" || sourceURL.User != nil {
		return false
	}

	return strings.EqualFold(sourceURL.Scheme, requestScheme(request)) &&
		equalOriginHost(sourceURL, requestScheme(request), request.Host)
}

func requestScheme(request *http.Request) string {
	if request.TLS != nil {
		return "https"
	}

	forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0])
	if forwarded != "" {
		return strings.ToLower(forwarded)
	}

	return "http"
}

func equalOriginHost(source *url.URL, scheme, requestHost string) bool {
	return strings.EqualFold(canonicalHost(source.Scheme, source.Host), canonicalHost(scheme, requestHost))
}

func canonicalHost(scheme, host string) string {
	parsed, err := url.Parse("//" + host)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}

	port := parsed.Port()
	if port == "" {
		switch strings.ToLower(scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	return strings.ToLower(parsed.Hostname()) + ":" + port
}
