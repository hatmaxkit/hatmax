// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package intent

import "hatmax.adrianpk.com/generator/book"

// Status classifies intent admission or rejection.
type Status string

const (
	// StatusAdmitted means the intent is ready for deterministic planning.
	StatusAdmitted Status = "admitted"
	// StatusClarificationRequired means material product decisions are missing.
	StatusClarificationRequired Status = "clarification_required"
	// StatusInvalid means the typed intent violates its schema.
	StatusInvalid Status = "intent_invalid"
	// StatusIncompatible means the intent conflicts with project or Book state.
	StatusIncompatible Status = "intent_incompatible"
	// StatusCapabilityUnsupported means a requested capability is unavailable.
	StatusCapabilityUnsupported Status = "capability_unsupported"
	// StatusExceptionRequired means an exception is invalid or cannot execute.
	StatusExceptionRequired Status = "exception_required"
)

// Diagnostic describes one stable semantic validation failure.
type Diagnostic struct {
	Code    string `json:"code" yaml:"code"`
	Field   string `json:"field" yaml:"field"`
	Message string `json:"message" yaml:"message"`
}

// Clarification asks for one product decision without exposing architecture
// choices owned by the Book.
type Clarification struct {
	Field    string `json:"field" yaml:"field"`
	Question string `json:"question" yaml:"question"`
}

// Result is the deterministic outcome of intent validation.
type Result struct {
	Status         Status
	Intent         Intent
	Selection      book.Selection
	Diagnostics    []Diagnostic
	Clarifications []Clarification
}

// Admitted reports whether deterministic planning may continue.
func (r Result) Admitted() bool {
	return r.Status == StatusAdmitted
}
