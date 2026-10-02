// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

import (
	"context"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

type requestIPKey struct{}

type requestIP struct {
	peer   string
	client string
}

// ProxyHeaders records the connection peer and resolves the client IP through
// explicitly trusted proxy networks. With no networks, forwarded headers are
// ignored. Install it before RateLimit and any middleware that rewrites RemoteAddr.
// RemoteAddr is never changed; InternalOnly always checks the connection peer.
func ProxyHeaders(trustedProxies ...netip.Prefix) func(http.Handler) http.Handler {
	trusted := slices.Clone(trustedProxies)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer := connectionPeer(r)
			identity := requestIP{peer: peer, client: forwardedClient(r, peer, trusted)}
			ctx := context.WithValue(r.Context(), requestIPKey{}, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClientIP returns the canonical IP resolved by ProxyHeaders, or the connection
// peer when that middleware is absent. It never trusts headers on its own.
// An unparseable RemoteAddr is returned unchanged.
func ClientIP(r *http.Request) string {
	identity, ok := r.Context().Value(requestIPKey{}).(requestIP)

	if ok {
		return identity.client
	}

	return connectionPeer(r)
}

func connectionPeer(r *http.Request) string {
	identity, ok := r.Context().Value(requestIPKey{}).(requestIP)

	if ok {
		return identity.peer
	}

	ip := parsePeer(r.RemoteAddr)

	if ip.IsValid() {
		return ip.String()
	}

	return r.RemoteAddr
}

func parsePeer(remoteAddr string) netip.Addr {
	addr, err := netip.ParseAddrPort(remoteAddr)
	if err == nil {
		return addr.Addr().WithZone("").Unmap()
	}

	ip, err := netip.ParseAddr(remoteAddr)
	if err != nil {
		return netip.Addr{}
	}

	return ip.WithZone("").Unmap()
}

func trustedIP(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}

func parseForwardedIP(value string) netip.Addr {
	ip, err := netip.ParseAddr(strings.TrimSpace(value))

	if err != nil || ip.Zone() != "" {
		return netip.Addr{}
	}

	return ip.Unmap()
}

func forwardedClient(r *http.Request, peer string, trusted []netip.Prefix) string {
	ip := parsePeer(peer)

	if !trustedIP(ip, trusted) {
		return peer
	}

	values := r.Header.Values("X-Forwarded-For")

	if len(values) > 0 {
		// Walk from the socket towards the client; anything before the first
		// untrusted hop is controlled by that hop, not by our approved proxies.
		for _, value := range slices.Backward(values) {
			hops := strings.Split(value, ",")

			for _, hop := range slices.Backward(hops) {
				if !trustedIP(ip, trusted) {
					return ip.String()
				}

				ip = parseForwardedIP(hop)

				if !ip.IsValid() {
					return peer
				}
			}
		}

		return ip.String()
	}

	values = r.Header.Values("X-Real-IP")

	if len(values) == 1 {
		ip = parseForwardedIP(values[0])

		if ip.IsValid() {
			return ip.String()
		}
	}

	return peer
}
