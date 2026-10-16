// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

type root struct {
	Group   string
	Package string
	Name    string
}

func (r root) key() string { return r.Group + ":" + r.Package + ":" + r.Name }

type criterion struct {
	ID    string
	Roots []root
}

// This versioned ledger binds criteria to actual roots; runtime receipts bind
// its exact source bytes and candidate. OS10 also requires the full gate mode.
var criteria = []criterion{
	{"OS01", []root{{"ordinary", "storage/local", "TestObjects"}, {"remote", "storage/s3", "TestPrivateObjects"}, {"remote", "storage/s3", "TestObjectLifecycle"}}},
	{"OS02", []root{{"ordinary", "storage", "TestKeys"}, {"ordinary", "storage/local", "TestConfinement"}}},
	{"OS03", []root{{"ordinary", "storage", "TestBounds"}, {"ordinary", "storage/local", "TestWriteFailure"}, {"remote", "storage/s3", "TestRemoteBounds"}}},
	{"OS04", []root{{"ordinary", "storage/s3", "TestProviderErrors"}, {"remote", "storage/s3", "TestRemoteFailures"}}},
	{"OS05", []root{{"remote", "storage/s3", "TestSignedUploads"}}},
	{"OS06", []root{{"ordinary", "storage/s3", "TestSigningClock"}, {"ordinary", "storage/s3", "TestSigningCredentials"}, {"remote", "storage/s3", "TestSigningExpiry"}}},
	{"OS07", []root{{"remote", "storage/s3", "TestSnapshotCompletion"}, {"remote", "storage/s3", "TestObjectLifecycle"}}},
	{"OS08", []root{{"ordinary", "scripts/documentation-conformance", "TestObjectImageCompatibility"}, {"ordinary", "image/s3", "TestUploadInput"}, {"ordinary", "image/stdprocessor", "TestProcessorResize"}}},
	{"OS09", []root{{"ordinary", "scripts/documentation-conformance", "TestObjectGuideDrift"}, {"ordinary", "scripts/documentation-conformance", "TestObjectLocalGuide"}, {"guide", "scripts/documentation-conformance", "TestObjectStorageGuide"}, {"guide-child", "storage/s3", "TestDocumentedStorage"}}},
	{"OS10", []root{{"ordinary", "scripts/documentation-conformance", "TestObjectAcceptance"}, {"ordinary", "scripts/object-storage", "TestEvidenceControls"}}},
}

const fixtureSource = "9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a"
const fixtureModule = "v0.0.0-20251015172955-9e49d5e7a648"
const fixtureSum = "h1:6TdolSCLSs2nwm8i0PpWDqf9iX2Ty9WQK8wmr7dCnUM="

// Ordering is part of acceptance, independently of the runtime step files.
func expectedLabels(mode string) []string {
	labels := []string{
		"evidence tool build", "storage formatting", "source licensing",
		"runner shell syntax", "fixture tool shell syntax", "PostgreSQL fixture shell syntax",
		"ordinary storage and image adapter roots", "race storage and image adapter roots",
		"affected build", "affected vet", "configured affected lint",
		"private S3 integration roots", "integration vet", "configured integration lint",
		"documentation structure, links and compiled examples", "published actual signed remote workflow",
		"documented storage child results", "documentation integration vet", "configured documentation integration lint",
	}
	if mode == "full" {
		labels = append(labels, "complete repository gate")
	} else {
		labels = append(labels, "native PostgreSQL smoke")
	}

	return append(labels, "owned PostgreSQL cleanup")
}
