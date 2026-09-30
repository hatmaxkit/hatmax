// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package middleware

import (
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5/middleware"
)

// DefaultStack returns the standard middleware stack for all hatmax services.
// This stack includes: RequestID, ProxyHeaders, Logger, and Recoverer.
// Forwarded client IPs are accepted only from the supplied trusted networks.
func DefaultStack(trustedProxies ...netip.Prefix) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		RequestID,
		ProxyHeaders(trustedProxies...),
		middleware.Logger,
		middleware.Recoverer,
	}
}

// DefaultInternal returns the standard middleware stack plus InternalOnly restriction.
// Use this for internal services that should only be accessible from private networks.
func DefaultInternal(trustedProxies ...netip.Prefix) []func(http.Handler) http.Handler {
	stack := DefaultStack(trustedProxies...)

	return append(stack, InternalOnly())
}

// InternalOnly restricts access to internal networks only.
// It checks the connection peer, not a proxy-derived client IP.
// This provides defense-in-depth and complements (does not replace) network policies
// at the infrastructure level.
func InternalOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isInternalIP(connectionPeer(r)) {
				http.Error(w, "Forbidden", http.StatusForbidden)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isInternalIP(remoteAddr string) bool {
	ip := parsePeer(remoteAddr)

	return ip.IsLoopback() || ip.IsPrivate()
}
