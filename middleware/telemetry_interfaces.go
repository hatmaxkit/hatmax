// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

// RequestCounter increments on each HTTP request.
type RequestCounter interface {
	IncrementRequests()
}

// CrashRecorder records panic information.
type CrashRecorder interface {
	RecordPanic(message, endpoint, method string)
}
