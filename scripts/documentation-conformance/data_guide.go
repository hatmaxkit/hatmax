// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (v *verification) dataGuide(discovered inventory) error {
	directory := filepath.Join(v.fixture, "data-guide")

	err := writeFile(filepath.Join(directory, "config.yaml"), discovered.contents["examples/guide/config.yaml"])
	if err != nil {
		return err
	}

	binary := filepath.Join(directory, "guide")

	err = v.exec(v.root, "go", "build", "-o", binary, "./examples/guide")
	if err != nil {
		return err
	}

	for attempt := 1; attempt <= 2; attempt++ {
		err = v.guideDataAttempt(binary, directory, attempt)
		if err != nil {
			return err
		}
	}

	v.receipt.Observations = append(v.receipt.Observations, "Guide database mode: matching-origin note submission persists title and slug across process restart; greeting changes to Welcome and resets to Hello at restart; empty note rejected; safe name-validation feedback and escaped HTML; owned listeners stopped")

	return nil
}

func (v *verification) guideDataAttempt(binary, directory string, attempt int) (result error) {
	address, err := freeAddress()
	if err != nil {
		return err
	}

	environment := []string{
		"GUIDE_DATABASE_ENABLED=true", "GUIDE_SERVER_PORT=" + address,
		"GUIDE_DATABASE_HOST=" + os.Getenv("DB_HOST"), "GUIDE_DATABASE_PORT=" + os.Getenv("DB_PORT"),
		"GUIDE_DATABASE_USER=" + os.Getenv("DB_USER"), "GUIDE_DATABASE_PASSWORD=" + os.Getenv("DB_PASSWORD"),
		"GUIDE_DATABASE_DATABASE=" + os.Getenv("DB_NAME"), "GUIDE_DATABASE_SCHEMA=guide_data",
	}

	process, err := v.start(binary, directory, environment, fmt.Sprintf("guide-data-%02d", attempt))
	if err != nil {
		return err
	}

	defer func() {
		err := process.stop(false)
		if result == nil {
			result = err
		}

		_, _, _, requestErr := request(localClient(), "GET", "http://"+address+"/ping", "", nil)
		if result == nil && requestErr == nil {
			result = fmt.Errorf("owned Guide listener remains available after stop")
		}
	}()

	client := localClient()
	endpoint := "http://" + address

	err = awaitHealth(client, endpoint)
	if err != nil {
		return err
	}

	headers := map[string]string{"Origin": endpoint, "Content-Type": "application/x-www-form-urlencoded"}

	status, _, output, err := request(client, "GET", endpoint+"/greeting", "", nil)
	if err != nil || status != http.StatusOK || strings.TrimSpace(output) != "Hello" {
		return fmt.Errorf("Guide greeting default/restart failed: %v", err)
	}

	if attempt == 1 {
		status, responseHeaders, _, err := request(client, "POST", endpoint+"/notes", url.Values{"title": {"Slice 3 note"}}.Encode(), headers)
		if err != nil || status != http.StatusSeeOther || responseHeaders.Get("Location") != "/" {
			return fmt.Errorf("Guide durable note submission failed: %v", err)
		}
	}

	status, _, output, err = request(client, "GET", endpoint+"/", "", nil)
	if err != nil || status != http.StatusOK || !strings.Contains(output, "Slice 3 note") || !strings.Contains(output, "slice-3-note-") {
		return fmt.Errorf("Guide note persistence/slug failed: %v", err)
	}

	status, _, _, err = request(client, "POST", endpoint+"/notes", "title=", headers)
	if err != nil || status != http.StatusBadRequest {
		return fmt.Errorf("Guide empty note was accepted: %v", err)
	}

	status, responseHeaders, _, err := request(client, "POST", endpoint+"/greeting", "greeting=Welcome", headers)
	if err != nil || status != http.StatusSeeOther || responseHeaders.Get("Location") != "/" {
		return fmt.Errorf("Guide greeting update failed: %v", err)
	}

	status, _, output, err = request(client, "GET", endpoint+"/greeting", "", nil)
	if err != nil || status != http.StatusOK || strings.TrimSpace(output) != "Welcome" {
		return fmt.Errorf("Guide greeting read after update failed: %v", err)
	}

	for _, tc := range []struct {
		name, expected string
	}{
		{"Alice", "Accepted Alice"}, {"A", "must be at least"}, {"<b>Alice</b>", "&lt;b&gt;Alice&lt;/b&gt;"},
	} {
		status, _, output, err = request(client, "POST", endpoint+"/name", url.Values{"name": {tc.name}}.Encode(), headers)
		if err != nil || status != http.StatusOK || !strings.Contains(output, tc.expected) {
			return fmt.Errorf("Guide validation/escaping failed: %v", err)
		}
	}

	return nil
}
