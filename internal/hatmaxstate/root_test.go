// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package hatmaxstate

import (
	"path/filepath"
	"testing"
)

func TestResolveRootUsesPlatformUserStateLocations(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		home         string
		localAppData string
		xdg          string
		want         string
	}{
		{
			name: "linux xdg",
			goos: "linux",
			home: "/home/adrian",
			xdg:  "/state",
			want: filepath.Join("/state", "hatmax"),
		},
		{
			name: "linux fallback",
			goos: "linux",
			home: "/home/adrian",
			xdg:  "relative/state",
			want: filepath.Join("/home/adrian", ".local", "state", "hatmax"),
		},
		{
			name: "macos",
			goos: "darwin",
			home: "/Users/adrian",
			want: filepath.Join("/Users/adrian", "Library", "Application Support", "Hatmax", "State"),
		},
		{
			name:         "windows",
			goos:         "windows",
			localAppData: `C:\Users\adrian\AppData\Local`,
			want:         filepath.Join(`C:\Users\adrian\AppData\Local`, "Hatmax", "State"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveRoot(test.goos, test.home, test.localAppData, test.xdg)
			if err != nil {
				t.Fatalf("ResolveRoot() error = %v", err)
			}

			if got != test.want {
				t.Errorf("ResolveRoot() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveRootRejectsUnavailableUserState(t *testing.T) {
	_, err := ResolveRoot("linux", "", "", "relative")
	requireStateCode(t, err, "state_root_unavailable")

	_, err = ResolveRoot("darwin", "", "", "")
	requireStateCode(t, err, "state_root_unavailable")

	_, err = ResolveRoot("windows", "", "", "")
	requireStateCode(t, err, "state_root_unavailable")
}
