// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package intent

import (
	"errors"
	"fmt"

	"hatmax.adrianpk.com/generator/book"
)

func validateCapabilities(
	value Intent,
	selectedBook *book.Book,
	archetype book.Archetype,
	operation book.Operation,
) (book.Selection, *Result) {
	requested := make(map[string]struct{}, len(value.Capabilities))
	allowed := make(map[string]struct{}, len(archetype.RequiredCapabilities)+len(archetype.OptionalCapabilities))

	for _, capability := range archetype.RequiredCapabilities {
		allowed[capability] = struct{}{}
	}

	for _, capability := range archetype.OptionalCapabilities {
		allowed[capability] = struct{}{}
	}

	for _, capability := range value.Capabilities {
		requested[capability] = struct{}{}

		definition, exists := selectedBook.Capability(capability)
		if !exists {
			return book.Selection{}, capabilityFailure(
				value,
				"HMGEN-CAPABILITY-UNSUPPORTED",
				fmt.Sprintf("capability %q is not defined by the selected Book", capability),
			)
		}

		if _, admitted := allowed[capability]; !admitted {
			return book.Selection{}, capabilityFailure(
				value,
				"HMGEN-CAPABILITY-UNSUPPORTED",
				fmt.Sprintf("capability %q is not admitted by archetype %q", capability, value.Archetype),
			)
		}

		for _, prerequisite := range definition.Requires {
			if !containsCapability(value.Capabilities, prerequisite) {
				return book.Selection{}, incompatibleCapabilityFailure(
					value,
					"HMGEN-CAPABILITY-PREREQUISITE",
					fmt.Sprintf("capability %q requires %q", capability, prerequisite),
				)
			}
		}
	}

	required := append([]string{}, archetype.RequiredCapabilities...)

	required = append(required, operation.RequiredCapabilities...)
	for _, capability := range required {
		if value.Operation == OperationCreateApplication {
			continue
		}

		if _, exists := requested[capability]; !exists {
			return book.Selection{}, incompatibleCapabilityFailure(
				value,
				"HMGEN-CAPABILITY-REQUIRED",
				fmt.Sprintf("operation %q requires capability %q", value.Operation, capability),
			)
		}
	}

	selection, err := selectedBook.Select(value.Archetype, value.Capabilities)
	if err != nil {
		var validationErr book.ValidationError
		if errors.As(err, &validationErr) && validationErr.Code == "book_capability_conflict" {
			failure := incompatibleCapabilityFailure(value, "HMGEN-CAPABILITY-CONFLICT", validationErr.Message)

			return book.Selection{}, failure
		}

		failure := incompatibleCapabilityFailure(value, "HMGEN-CAPABILITY-SELECTION", err.Error())

		return book.Selection{}, failure
	}

	return selection, nil
}

func containsCapability(capabilities []string, expected string) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}

	return false
}

func capabilityFailure(value Intent, code, message string) *Result {
	return &Result{
		Status: StatusCapabilityUnsupported,
		Intent: value,
		Diagnostics: []Diagnostic{{
			Code:    code,
			Field:   "capabilities",
			Message: message,
		}},
	}
}

func incompatibleCapabilityFailure(value Intent, code, message string) *Result {
	return &Result{
		Status: StatusIncompatible,
		Intent: value,
		Diagnostics: []Diagnostic{{
			Code:    code,
			Field:   "capabilities",
			Message: message,
		}},
	}
}
