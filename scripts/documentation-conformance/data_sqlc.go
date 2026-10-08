// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"path/filepath"
	"strings"
)

func (v *verification) tickedSQLC(discovered inventory) error {
	directory := filepath.Join(v.fixture, "ticked-sqlc")

	err := v.module(directory)
	if err != nil {
		return err
	}

	for file, contents := range discovered.contents {
		relative := strings.TrimPrefix(file, "examples/ticked/")
		if relative == file || (relative != "sqlc.yaml" && !strings.HasPrefix(relative, "db/queries/") && !strings.HasPrefix(relative, "assets/migration/postgres/")) {
			continue
		}

		err = writeFile(filepath.Join(directory, relative), contents)
		if err != nil {
			return err
		}
	}

	err = v.exec(directory, "sqlc", "generate")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "build", "./...")
	if err != nil {
		return err
	}

	return v.exec(v.root, "go", "build", "./examples/ticked/internal/dal")
}
