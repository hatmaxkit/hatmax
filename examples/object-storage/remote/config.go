// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"context"
	"os"
	"time"

	"hatmax.adrianpk.com/storage/s3"
)

const (
	maximum    = 1 << 20
	stagingKey = "guide/staging-document.txt"
	finalKey   = "guide/final-document.txt"
)

func newStore(ctx context.Context) (*s3.Store, error) {
	return s3.NewStore(ctx, s3.Options{
		Bucket:         os.Getenv("OBJECT_STORAGE_BUCKET"),
		Region:         os.Getenv("OBJECT_STORAGE_REGION"),
		Endpoint:       os.Getenv("OBJECT_STORAGE_ENDPOINT"),
		SpoolDirectory: os.Getenv("OBJECT_STORAGE_SPOOL"),
		MaxSpoolBytes:  maximum,
		ReadAttempts:   1, OperationTimeout: 30 * time.Second,
		PostPolicy: s3.PostPolicyMinIO,
		// Enable only for an owned literal-loopback fixture, never to relax TLS.
		AllowInsecureLoopback: os.Getenv("OBJECT_STORAGE_FIXTURE_LOOPBACK") == "1",
	})
}
