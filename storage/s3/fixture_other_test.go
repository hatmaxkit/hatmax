//go:build integration && !linux

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"errors"
	"os/exec"
)

func startFixtureProcess(*exec.Cmd) (chan error, error) {
	return nil, errors.New("native fixture process ownership requires Linux")
}
