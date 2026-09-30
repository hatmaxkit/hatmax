// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import "sort"

// ChangeKind describes how one relevant observation drifted.
type ChangeKind string

const (
	// ChangeAdded means a relevant observation appeared.
	ChangeAdded ChangeKind = "added"
	// ChangeModified means a relevant observation changed.
	ChangeModified ChangeKind = "modified"
	// ChangeRemoved means a relevant observation disappeared.
	ChangeRemoved ChangeKind = "removed"
)

// Change describes one relevant difference between two fingerprints.
type Change struct {
	Kind         ChangeKind
	Class        ObservationClass
	Path         string
	Surface      string
	BeforeDigest string
	AfterDigest  string
}

// CompareFingerprints classifies relevant drift between two fingerprints.
func CompareFingerprints(before, after Fingerprint) []Change {
	beforeByIdentity := indexObservations(before.Observations)
	afterByIdentity := indexObservations(after.Observations)
	identities := make(map[string]struct{}, len(beforeByIdentity)+len(afterByIdentity))

	for identity := range beforeByIdentity {
		identities[identity] = struct{}{}
	}

	for identity := range afterByIdentity {
		identities[identity] = struct{}{}
	}

	result := make([]Change, 0)

	for identity := range identities {
		previous, existedBefore := beforeByIdentity[identity]
		current, existsAfter := afterByIdentity[identity]

		switch {
		case !existedBefore:
			result = append(result, changeFromObservation(ChangeAdded, current, "", current.Digest))
		case !existsAfter:
			result = append(result, changeFromObservation(ChangeRemoved, previous, previous.Digest, ""))
		case previous.Digest != current.Digest:
			result = append(result, changeFromObservation(ChangeModified, current, previous.Digest, current.Digest))
		}
	}

	sort.Slice(result, func(left, right int) bool {
		if result[left].Class != result[right].Class {
			return result[left].Class < result[right].Class
		}

		if result[left].Surface != result[right].Surface {
			return result[left].Surface < result[right].Surface
		}

		if result[left].Path != result[right].Path {
			return result[left].Path < result[right].Path
		}

		return result[left].Kind < result[right].Kind
	})

	return result
}

func indexObservations(observations []Observation) map[string]Observation {
	result := make(map[string]Observation, len(observations))
	for _, observation := range observations {
		result[observationIdentity(observation)] = observation
	}

	return result
}

func changeFromObservation(kind ChangeKind, observation Observation, before, after string) Change {
	return Change{
		Kind:         kind,
		Class:        observation.Class,
		Path:         observation.Path,
		Surface:      observation.Surface,
		BeforeDigest: before,
		AfterDigest:  after,
	}
}
