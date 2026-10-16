<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Configure Private S3 Object Storage

Use `storage/s3` when the application needs private Put/Open/Stat/Delete without
image dimensions or URL helpers. Provision an existing private, initially
unversioned bucket and an application-controlled spool directory. The
constructor creates neither and does not probe bucket access.

1. Choose the bucket, region, endpoint and credentials in deployment configuration.
   Never accept them from an upload request. Leave `Credentials` unset for the
   SDK's default renewable credential chain, or supply an `aws.CredentialsProvider`.
2. Create the spool directory with mode 0700. Set `MaxSpoolBytes` to a finite
   per-write capacity and admit each write's `MaxBytes` within it. N concurrent
   writes can use N times that capacity plus filesystem overhead; callers own
   aggregate quotas and admission. Copies use fixed 32 KiB buffers.
3. Use verified HTTPS. `AllowInsecureLoopback` permits HTTP only for a literal
   loopback address; it never disables certificate verification for HTTPS.
4. Choose a finite operation timeout and read-attempt limit. Close returned
   readers before closing the store. Callers retain ownership of input readers.

The [remote executable](../../../examples/object-storage/remote/config.go)
uses this exact configuration, with a 1 MiB `maximum`. Its POST profile is enabled
for the signing procedure; I/O-only callers leave `PostPolicy` disabled.

<!-- object-storage-source: examples/object-storage/remote/config.go#newStore -->
```go
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
```

`PostPolicyMinIO` asserts that the explicitly configured endpoint matches the
tested unversioned MinIO `RELEASE.2025-10-15T17-29-55Z` profile. It is a deployment
assertion, not vendor detection. An arbitrary compatible endpoint is not proven
by setting the value. Unknown/disabled signing capabilities fail explicitly;
there is no weaker presigned PUT fallback.

Verify construction with the [signed-upload walkthrough](../sign-object-uploads/README.md)
in an isolated bucket owned by the exercise. The repository runner uses its own
native loopback server and ephemeral credentials. Keep credentials in the SDK's
supported credential mechanisms; do not print them or embed them in commands.
Delete removes the current key in the supported unversioned profile; it does
not purge historical versions in a versioned bucket.

See the [exact options and error reference](../../reference/storage/README.md)
and [mutation reconciliation](../reconcile-object-storage/README.md).
