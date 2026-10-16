//go:build integration && documentation

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Run the published example against the same identified actual server fixture.
func TestDocumentedStorage(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	binary := filepath.Join(t.TempDir(), "remote-guide")
	build := exec.CommandContext(ctx, "go", "build", "-race", "-o", binary, "./examples/object-storage/remote")
	build.Dir = root

	_, err = build.CombinedOutput()
	if err != nil {
		t.Fatalf("build documented remote example: %v", err)
	}

	material, err := live.provider.Retrieve(ctx)
	if err != nil {
		t.Fatal("owned fixture credentials unavailable")
	}

	spool := t.TempDir()
	command := exec.CommandContext(ctx, binary)
	command.WaitDelay = 5 * time.Second

	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "AWS_") && !strings.HasPrefix(value, "OBJECT_STORAGE_") {
			command.Env = append(command.Env, value)
		}
	}

	command.Env = append(command.Env,
		"OBJECT_STORAGE_BUCKET="+live.bucket,
		"OBJECT_STORAGE_REGION=us-east-1",
		"OBJECT_STORAGE_ENDPOINT="+live.endpoint,
		"OBJECT_STORAGE_SPOOL="+spool,
		"OBJECT_STORAGE_FIXTURE_LOOPBACK=1",
		"AWS_ACCESS_KEY_ID="+material.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY="+material.SecretAccessKey,
		"AWS_EC2_METADATA_DISABLED=true",
		"AWS_CONFIG_FILE="+filepath.Join(t.TempDir(), "absent"),
		"AWS_SHARED_CREDENTIALS_FILE="+filepath.Join(t.TempDir(), "absent"))

	output, err := command.CombinedOutput()
	if err != nil || string(output) != "remote object workflow complete\n" {
		// Captured output is never included in diagnostics; it may contain a grant.
		t.Fatal("documented remote command failed or emitted unexpected output")
	}

	objects, err := live.admin.ListObjectsV2(ctx, &sdk.ListObjectsV2Input{Bucket: aws.String(live.bucket), Prefix: aws.String("guide/"), MaxKeys: aws.Int32(1)})
	if err != nil || len(objects.Contents) != 0 || aws.ToBool(objects.IsTruncated) {
		t.Fatal("documented remote command did not remove its owned objects")
	}

	assertEmptySpool(t, spool)
	live.record("guide: signed POST and exact final readback passed\nguide: spool empty\n")
}
