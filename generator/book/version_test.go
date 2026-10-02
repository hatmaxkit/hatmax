// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package book

import "testing"

func TestSupportsHatmaxUsesHalfOpenVersionRange(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{version: "0.3.999", want: false},
		{version: "0.4.0", want: true},
		{version: "v0.4.0", want: true},
		{version: "0.4.999", want: true},
		{version: "0.5.0", want: true},
		{version: "v0.5.0", want: true},
		{version: "0.5.999", want: true},
		{version: "0.6.0", want: false},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			got, supportErr := loaded.SupportsHatmax(test.version)
			if supportErr != nil {
				t.Fatalf("SupportsHatmax(%q) error = %v", test.version, supportErr)
			}

			if got != test.want {
				t.Errorf("SupportsHatmax(%q) = %t, want %t", test.version, got, test.want)
			}
		})
	}
}

func TestSupportsHatmaxRejectsMalformedVersion(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	_, err = loaded.SupportsHatmax("0.4")
	if err == nil {
		t.Fatal("SupportsHatmax() error = nil, want malformed version error")
	}
}
