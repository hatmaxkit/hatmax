// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"time"

	"hatmax.adrianpk.com/storage"
)

func submit(ctx context.Context, grant storage.UploadGrant) error {
	// This example buffers one fixed small document and bounded signer fields.
	var body bytes.Buffer

	form := multipart.NewWriter(&body)

	fields := make([]string, 0, len(grant.Fields))
	for name := range grant.Fields {
		fields = append(fields, name)
	}

	slices.Sort(fields)

	for _, name := range fields {
		err := form.WriteField(name, grant.Fields[name])
		if err != nil {
			return err
		}
	}

	part, err := form.CreateFormFile("file", "document.txt")
	if err != nil {
		return err
	}

	_, err = io.WriteString(part, "Approved document\nPrivate bytes without image metadata.\n")
	if err != nil {
		return err
	}

	err = form.Close()
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, grant.URL, bytes.NewReader(body.Bytes()))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", form.FormDataContentType())

	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()

	client := &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 4096))

	err = errors.Join(readErr, response.Body.Close())
	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("signed POST rejected: HTTP %d", response.StatusCode)
	}

	return nil
}
