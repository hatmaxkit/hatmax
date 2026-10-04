// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/protocol"
)

// MaxEnrollmentBody is enforced before any JSON or CBOR parsing.
const MaxEnrollmentBody = 64 * 1024

func boundedJSON(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	depth := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}

		if err != nil {
			return ErrEnrollment
		}

		delimiter, ok := token.(json.Delim)
		if !ok {
			continue
		}

		switch delimiter {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}

		if depth > 16 || depth < 0 {
			return ErrEnrollment
		}
	}

	if depth != 0 {
		return ErrEnrollment
	}

	return nil
}

func decodeCeremonyPart(value string, limit int) ([]byte, error) {
	if len(value) > base64.RawURLEncoding.EncodedLen(limit) {
		return nil, ErrEnrollment
	}

	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) == 0 || len(decoded) > limit {
		return nil, ErrEnrollment
	}

	return decoded, nil
}

func parseEnrollmentResponse(body []byte) (*protocol.ParsedCredentialCreationData, error) {
	if len(body) == 0 || len(body) > MaxEnrollmentBody {
		return nil, ErrEnrollment
	}

	err := boundedJSON(body)
	if err != nil {
		return nil, err
	}

	var wire struct {
		ID       string `json:"id"`
		RawID    string `json:"rawId"`
		Response struct {
			ClientData  string `json:"clientDataJSON"`
			Attestation string `json:"attestationObject"`
		} `json:"response"`
	}

	err = json.Unmarshal(body, &wire)
	if err != nil || wire.ID != wire.RawID {
		return nil, ErrEnrollment
	}

	_, err = decodeCeremonyPart(wire.RawID, 1024)
	if err != nil {
		return nil, err
	}

	client, err := decodeCeremonyPart(wire.Response.ClientData, 8192)
	if err != nil {
		return nil, err
	}

	err = boundedJSON(client)
	if err != nil {
		return nil, err
	}

	var clientData struct {
		CrossOrigin bool            `json:"crossOrigin"`
		TopOrigin   json.RawMessage `json:"topOrigin"`
	}

	err = json.Unmarshal(client, &clientData)
	if err != nil || clientData.CrossOrigin || len(clientData.TopOrigin) != 0 {
		return nil, ErrEnrollment
	}

	attestation, err := decodeCeremonyPart(wire.Response.Attestation, 32768)
	if err != nil {
		return nil, err
	}

	mode, err := (cbor.DecOptions{MaxNestedLevels: 16, MaxArrayElements: 64, MaxMapPairs: 64, DupMapKey: cbor.DupMapKeyEnforcedAPF, IndefLength: cbor.IndefLengthForbidden, TagsMd: cbor.TagsForbidden}).DecMode()
	if err != nil {
		return nil, err
	}

	err = mode.Wellformed(attestation)
	if err != nil {
		return nil, ErrEnrollment
	}

	var object struct {
		AuthData []byte `cbor:"authData"`
		Format   string `cbor:"fmt"`
	}

	err = mode.Unmarshal(attestation, &object)
	if err != nil || object.Format != "none" || len(object.AuthData) > 8192 {
		return nil, ErrEnrollment
	}
	// The attested credential structure precedes the COSE key. Validate its ID
	// and bound the entire remainder before the library decodes the key/extensions.
	if len(object.AuthData) < 55 {
		return nil, ErrEnrollment
	}

	idLength := int(object.AuthData[53])<<8 | int(object.AuthData[54])
	if idLength == 0 || idLength > 1024 || len(object.AuthData) <= 55+idLength || len(object.AuthData)-55-idLength > 4096 {
		return nil, ErrEnrollment
	}

	err = mode.Wellformed(object.AuthData[55+idLength:])
	if err != nil {
		return nil, ErrEnrollment
	}

	return protocol.ParseCredentialCreationResponseBytes(body)
}
