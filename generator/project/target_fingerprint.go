// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import (
	"fmt"
)

const (
	// ObservationTarget records parent, target, admission, and planned-path state.
	ObservationTarget ObservationClass = "target"
	// ObservationTargetEntry records one preserved or conflicting target entry.
	ObservationTargetEntry ObservationClass = "target_entry"
	// ObservationRemote records sanitized repository remote evidence.
	ObservationRemote ObservationClass = "remote"
)

// Fingerprint computes a stable target fingerprint for application planning.
func (i TargetInventory) Fingerprint(bookVersion int) (Fingerprint, error) {
	if bookVersion < 1 {
		return Fingerprint{}, projectError("project_book_version_invalid", "", "Book version must be positive")
	}

	observations := []Observation{{
		Key: "book:version", Class: ObservationBook, Digest: fmt.Sprintf("%d", bookVersion),
	}}

	targetDigest, err := digestValue(struct {
		Parent       string
		Target       string
		Exists       bool
		Empty        bool
		Admission    TargetAdmission
		PlannedPaths []string
	}{i.Parent, i.Target, i.Exists, i.Empty, i.Admission, i.PlannedPaths})
	if err != nil {
		return Fingerprint{}, err
	}

	observations = append(observations, Observation{
		Key: "target:state", Class: ObservationTarget, Path: i.Target, Digest: targetDigest,
	})

	for _, entry := range i.Entries {
		digest, digestErr := digestValue(entry)
		if digestErr != nil {
			return Fingerprint{}, digestErr
		}

		observations = append(observations, Observation{
			Key: "target:entry:" + entry.Path, Class: ObservationTargetEntry, Path: entry.Path, Digest: digest,
		})
	}

	for _, collision := range i.Collisions {
		digest, digestErr := digestValue(collision)
		if digestErr != nil {
			return Fingerprint{}, digestErr
		}

		observations = append(observations, Observation{
			Key:    "target:collision:" + collision.Path + ":" + collision.Reason,
			Class:  ObservationTarget,
			Path:   collision.Path,
			Digest: digest,
		})
	}

	for _, remote := range i.Remotes {
		digest, digestErr := digestValue(remote)
		if digestErr != nil {
			return Fingerprint{}, digestErr
		}

		observations = append(observations, Observation{
			Key: "remote:" + remote.Name, Class: ObservationRemote, Digest: digest,
		})
	}

	for _, instruction := range i.Rules.Instructions {
		digest, digestErr := digestTargetRule(instruction)
		if digestErr != nil {
			return Fingerprint{}, projectError("target_rule_read_failed", instruction, "%v", digestErr)
		}

		observations = append(observations, Observation{
			Key: "target:rule:" + instruction, Class: ObservationRepositoryRule, Path: instruction, Digest: digest,
		})
	}

	sortObservations(observations)

	value, err := digestObservations(observations)
	if err != nil {
		return Fingerprint{}, err
	}

	return Fingerprint{
		Value: value, BookVersion: bookVersion, SelectedPaths: cloneStrings(i.PlannedPaths), Observations: observations,
	}, nil
}

func digestTargetRule(path string) (string, error) {
	return digestTargetFile(path)
}
