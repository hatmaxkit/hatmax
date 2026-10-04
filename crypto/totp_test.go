// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package crypto

import (
	"strings"
	"testing"
)

func TestGenerateTOTPKey(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		account string
		wantErr bool
	}{
		{
			name:    "valid issuer and account",
			issuer:  "MyApp",
			account: "user@example.com",
			wantErr: false,
		},
		{
			name:    "empty issuer",
			issuer:  "",
			account: "user@example.com",
			wantErr: true,
		},
		{
			name:    "empty account",
			issuer:  "MyApp",
			account: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateTOTPKey(tt.issuer, tt.account)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTOTPKey() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if key == nil {
					t.Error("expected non-nil key")

					return
				}

				if key.Secret() == "" {
					t.Error("expected non-empty secret")
				}
			}
		})
	}
}

func TestGenerateQRCodePNG(t *testing.T) {
	key, err := GenerateTOTPKey("TestApp", "test@example.com")
	if err != nil {
		t.Fatalf("cannot generate test key: %v", err)
	}

	tests := []struct {
		name string
		key  interface {
			Image(int, int) (interface{ Bounds() interface{} }, error)
		}
		size    int
		wantErr bool
	}{
		{
			name:    "valid QR code generation",
			size:    200,
			wantErr: false,
		},
		{
			name:    "small size",
			size:    50,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateQRCodePNG(key, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateQRCodePNG() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if len(got) == 0 {
					t.Error("expected non-empty PNG data")
				}

				if !strings.HasPrefix(string(got[:8]), "\x89PNG") {
					t.Error("expected PNG header")
				}
			}
		})
	}
}
