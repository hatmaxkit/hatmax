package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// DefaultStack returns the standard middleware stack for all hatmax services.
// This stack includes: RequestID, RealIP, Logger, and Recoverer.
func DefaultStack() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
	}
}

// DefaultInternal returns the standard middleware stack plus InternalOnly restriction.
// Use this for internal services that should only be accessible from private networks.
func DefaultInternal() []func(http.Handler) http.Handler {
	stack := DefaultStack()

	return append(stack, InternalOnly())
}

// InternalOnly restricts access to internal networks only.
// This provides defense-in-depth and complements (does not replace) network policies
// at the infrastructure level.
func InternalOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isInternalIP(r.RemoteAddr) {
				http.Error(w, "Forbidden", http.StatusForbidden)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isInternalIP(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		if network.Contains(ip) {
			return true
		}
	}

	if strings.HasPrefix(host, "fc00:") || strings.HasPrefix(host, "fd00:") {
		return true
	}

	return false
}
