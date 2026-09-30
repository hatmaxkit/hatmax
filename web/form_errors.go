// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package web

import (
	"errors"

	"hatmax.adrianpk.com/validation"
)

// FormErrors contains safe, user-facing validation feedback for a form.
type FormErrors struct {
	General string
	Fields  map[string][]string
}

// NewFormErrors creates form errors with a general summary.
func NewFormErrors(general string) FormErrors {
	return FormErrors{
		General: general,
		Fields:  make(map[string][]string),
	}
}

// FormErrorsFrom maps structured validation errors from an error chain.
// The supplied general message is used without exposing the source error.
func FormErrorsFrom(err error, general string) FormErrors {
	formErrors := NewFormErrors(general)

	var validationErrors validation.ValidationErrors
	if errors.As(err, &validationErrors) {
		formErrors.addValidationErrors(validationErrors)
	}

	var validationErrorsPointer *validation.ValidationErrors
	if errors.As(err, &validationErrorsPointer) && validationErrorsPointer != nil {
		formErrors.addValidationErrors(*validationErrorsPointer)
	}

	return formErrors
}

// Add appends a user-facing message to a field or replaces the general message
// when field is empty.
func (e *FormErrors) Add(field, message string) {
	if message == "" {
		return
	}

	if field == "" {
		e.General = message

		return
	}

	if e.Fields == nil {
		e.Fields = make(map[string][]string)
	}

	e.Fields[field] = append(e.Fields[field], message)
}

// First returns the first message for a field.
func (e FormErrors) First(field string) string {
	messages := e.Fields[field]
	if len(messages) == 0 {
		return ""
	}

	return messages[0]
}

// For returns all messages for a field.
func (e FormErrors) For(field string) []string {
	return e.Fields[field]
}

// Has reports whether a field has at least one message.
func (e FormErrors) Has(field string) bool {
	return len(e.Fields[field]) > 0
}

// Any reports whether any general or field-specific feedback is present.
func (e FormErrors) Any() bool {
	if e.General != "" {
		return true
	}

	for _, messages := range e.Fields {
		if len(messages) > 0 {
			return true
		}
	}

	return false
}

func (e *FormErrors) addValidationErrors(validationErrors validation.ValidationErrors) {
	for _, validationError := range validationErrors {
		e.Add(validationError.Field, validationError.Message)
	}
}
