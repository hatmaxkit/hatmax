// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

import (
	"container/list"
	"errors"
	"net/http"
	"net/netip"
	"sync"
	"time"
)

// ErrRateLimitConfig rejects invalid finite peer admission limits.
var ErrRateLimitConfig = errors.New("invalid rate limit configuration")

// RateLimitConfig bounds one process-local fixed window per canonical peer.
type RateLimitConfig struct {
	Limit        int
	Window       time.Duration
	MaxPeers     int
	CleanupBatch int
}
type peerWindow struct {
	peer  string
	start time.Time
	count int
}

// RateLimiter owns finite counters and explicit bounded cleanup, without workers.
type RateLimiter struct {
	mu    sync.Mutex
	peers map[string]*list.Element
	order *list.List
	cfg   RateLimitConfig
	now   func() time.Time
}

// NewRateLimiter validates finite defaults and starts no goroutine.
func NewRateLimiter(cfg RateLimitConfig) (*RateLimiter, error) {
	if cfg.Limit == 0 {
		cfg.Limit = 12
	}

	if cfg.Window == 0 {
		cfg.Window = time.Minute
	}

	if cfg.MaxPeers == 0 {
		cfg.MaxPeers = 1024
	}

	if cfg.CleanupBatch == 0 {
		cfg.CleanupBatch = 128
	}

	if cfg.Limit < 1 || cfg.Limit > 1000 || cfg.Window < time.Second || cfg.Window > time.Hour || cfg.MaxPeers < 1 || cfg.MaxPeers > 10000 || cfg.CleanupBatch < 1 || cfg.CleanupBatch > 1000 {
		return nil, ErrRateLimitConfig
	}

	return &RateLimiter{peers: make(map[string]*list.Element), order: list.New(), cfg: cfg, now: time.Now}, nil
}
func (rl *RateLimiter) cleanupLocked(now time.Time) int {
	removed := 0

	for range min(rl.cfg.CleanupBatch, rl.order.Len()) {
		e := rl.order.Front()

		w := e.Value.(*peerWindow)
		if !now.Before(w.start.Add(rl.cfg.Window)) {
			delete(rl.peers, w.peer)
			rl.order.Remove(e)

			removed++
		} else {
			rl.order.MoveToBack(e)
		}
	}

	return removed
}

// Allow charges one fixed-window operation. Denial never extends a live window.
func (rl *RateLimiter) Allow(peer string) bool {
	if rl == nil || rl.now == nil {
		return false
	}

	ip, err := netip.ParseAddr(peer)
	if err != nil || ip.Zone() != "" {
		return false
	}

	peer = ip.Unmap().String()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	rl.cleanupLocked(now)

	e, exists := rl.peers[peer]
	if !exists {
		if len(rl.peers) >= rl.cfg.MaxPeers {
			return false
		}

		e = rl.order.PushBack(&peerWindow{peer: peer, start: now})
		rl.peers[peer] = e
	}

	w := e.Value.(*peerWindow)
	if !now.Before(w.start.Add(rl.cfg.Window)) {
		w.start, w.count = now, 0
	}

	if now.Before(w.start) || w.count >= rl.cfg.Limit {
		return false
	}

	w.count++

	return true
}

// Cleanup inspects at most one configured batch, retaining every live peer.
func (rl *RateLimiter) Cleanup() int {
	if rl == nil || rl.now == nil {
		return 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.cleanupLocked(rl.now())
}

// RateLimit uses the existing explicit trusted-proxy ClientIP contract.
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(ClientIP(r)) {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
