// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package middleware

// RequestCounter increments on each HTTP request.
type RequestCounter interface {
	IncrementRequests()
}

// CrashRecorder records panic information.
type CrashRecorder interface {
	RecordPanic(message, endpoint, method string)
}
