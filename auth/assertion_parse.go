// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"encoding/json"
	"github.com/go-webauthn/webauthn/protocol"
)

// Assertions contain no attested key or CBOR extension in the supported profile.
// Bounds are checked before the library decodes any protocol-controlled input.
func parseAssertionResponse(body []byte) (*protocol.ParsedCredentialAssertionData, error) {
	if len(body) == 0 || len(body) > MaxEnrollmentBody {
		return nil, ErrWebAuthn
	}

	err := boundedJSON(body)
	if err != nil {
		return nil, ErrWebAuthn
	}

	var wire struct {
		ID       string `json:"id"`
		RawID    string `json:"rawId"`
		Type     string `json:"type"`
		Response struct {
			Client        string `json:"clientDataJSON"`
			Authenticator string `json:"authenticatorData"`
			Signature     string `json:"signature"`
			Handle        string `json:"userHandle"`
		} `json:"response"`
	}

	err = json.Unmarshal(body, &wire)
	if err != nil || wire.ID != wire.RawID || wire.Type != "public-key" {
		return nil, ErrWebAuthn
	}

	_, err = decodeCeremonyPart(wire.RawID, 1024)
	if err != nil {
		return nil, ErrWebAuthn
	}

	client, err := decodeCeremonyPart(wire.Response.Client, 8192)
	if err != nil {
		return nil, ErrWebAuthn
	}

	err = boundedJSON(client)
	if err != nil {
		return nil, ErrWebAuthn
	}

	var origin struct {
		Cross bool            `json:"crossOrigin"`
		Top   json.RawMessage `json:"topOrigin"`
	}

	err = json.Unmarshal(client, &origin)
	if err != nil || origin.Cross || len(origin.Top) != 0 {
		return nil, ErrWebAuthn
	}

	data, err := decodeCeremonyPart(wire.Response.Authenticator, 37)
	if err != nil || len(data) != 37 || data[32]&0xc0 != 0 {
		return nil, ErrWebAuthn
	}

	_, err = decodeCeremonyPart(wire.Response.Signature, 80)
	if err != nil {
		return nil, ErrWebAuthn
	}

	if wire.Response.Handle != "" {
		handle, err := decodeCeremonyPart(wire.Response.Handle, 32)
		if err != nil || len(handle) != 32 {
			return nil, ErrWebAuthn
		}
	}

	return protocol.ParseCredentialRequestResponseBytes(body)
}
