// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"net/http"
	"time"
)

// unitIngress omits the response wait; real timing uses the integration boundary.
func unitIngress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry := ingressRequest{workDeadline: time.Now().Add(5 * time.Second), ackDeadline: time.Now()}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ingressKey{}, entry)))
	})
}
