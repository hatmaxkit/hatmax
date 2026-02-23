// Copyright 2024 The Hatmax Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testhelper

import (
	"os"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		want         string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR_SET",
			defaultValue: "default",
			envValue:     "custom",
			setEnv:       true,
			want:         "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR_NOT_SET",
			defaultValue: "default",
			setEnv:       false,
			want:         "default",
		},
		{
			name:         "env var empty",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "length 8",
			length: 8,
		},
		{
			name:   "length 16",
			length: 16,
		},
		{
			name:   "length 0",
			length: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomString(tt.length)
			if len(got) != tt.length {
				t.Errorf("randomString() length = %v, want %v", len(got), tt.length)
			}

			for _, c := range got {
				if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
					t.Errorf("randomString() contains invalid character: %c", c)
				}
			}
		})
	}
}

func TestRandomStringUniqueness(t *testing.T) {
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		s := randomString(8)
		if seen[s] {
			t.Errorf("randomString() produced duplicate: %s", s)
		}

		seen[s] = true
	}
}

func TestTestLogger(t *testing.T) {
	logger := TestLogger()
	if logger == nil {
		t.Error("TestLogger() returned nil")
	}
}
