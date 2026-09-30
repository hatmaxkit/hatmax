// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package plan

import "hatmax.adrianpk.com/generator/project"

// Diagnostic describes one stable plan lifecycle failure.
type Diagnostic struct {
	Code    string `json:"code" yaml:"code"`
	Field   string `json:"field" yaml:"field"`
	Message string `json:"message" yaml:"message"`
}

// Freshness reports whether current relevant project state still matches a
// sealed plan.
type Freshness struct {
	Stale       bool             `json:"stale" yaml:"stale"`
	Changes     []project.Change `json:"changes" yaml:"changes"`
	Diagnostics []Diagnostic     `json:"diagnostics" yaml:"diagnostics"`
}

// CheckFingerprint compares a sealed plan with a freshly computed project
// fingerprint without changing lifecycle state.
func CheckFingerprint(value Plan, current project.Fingerprint) (Freshness, error) {
	err := VerifyDigest(value)
	if err != nil {
		return Freshness{}, err
	}

	expectedFingerprint := value.ProjectFingerprint
	fingerprintField := "project_fingerprint"

	if value.SchemaVersion == ApplicationSchemaVersion {
		expectedFingerprint = value.SourceFingerprint
		fingerprintField = "source_fingerprint"
	}

	if current.Value == expectedFingerprint && current.BookVersion == value.BookVersion {
		return Freshness{
			Changes:     []project.Change{},
			Diagnostics: []Diagnostic{},
		}, nil
	}

	planned := project.Fingerprint{
		Value:        expectedFingerprint,
		BookVersion:  value.BookVersion,
		Observations: cloneObservations(value.ExpectedObservations),
	}
	changes := project.CompareFingerprints(planned, current)

	return Freshness{
		Stale:   true,
		Changes: changes,
		Diagnostics: []Diagnostic{{
			Code:    "HMGEN-PLAN-STALE",
			Field:   fingerprintField,
			Message: "relevant project state changed after the plan was produced",
		}},
	}, nil
}
