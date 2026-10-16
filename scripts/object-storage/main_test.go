// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import "testing"

// Missing or duplicate actual root identities must never satisfy acceptance.
func TestEvidenceControls(t *testing.T) {
	var roots []root

	seen := make(map[string]bool)

	for _, criterion := range criteria {
		for _, r := range criterion.Roots {
			if !seen[r.key()] {
				roots = append(roots, r)
				seen[r.key()] = true
			}
		}
	}

	for _, test := range []struct {
		name     string
		roots    []root
		rejected bool
	}{{"complete", roots, false}, {"missing root", roots[1:], true}, {"duplicate root", append(append([]root{}, roots...), roots[0]), true}} {
		t.Run(test.name, func(t *testing.T) {
			err := checkLedger(test.roots)
			if (err != nil) != test.rejected {
				t.Fatalf("ledger result: %v", err)
			}
		})
	}
}
