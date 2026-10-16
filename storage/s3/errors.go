// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"errors"

	"github.com/aws/smithy-go"

	"hatmax.adrianpk.com/storage"
)

func absent(err error) bool {
	var (
		api      smithy.APIError
		response interface{ HTTPStatusCode() int }
	)

	if !errors.As(err, &api) || !errors.As(err, &response) || response.HTTPStatusCode() != 404 {
		return false
	}

	return api.ErrorCode() == "NoSuchKey" || api.ErrorCode() == "NotFound"
}

func readError(err error) error {
	if absent(err) {
		return errors.Join(storage.ErrNotFound, err)
	}

	return err
}

func mutationError(err error) error {
	var response interface{ HTTPStatusCode() int }

	if errors.As(err, &response) {
		status := response.HTTPStatusCode()
		if status >= 400 && status < 500 && status != 408 {
			return err
		}
	}

	return errors.Join(storage.ErrIndeterminate, err)
}
