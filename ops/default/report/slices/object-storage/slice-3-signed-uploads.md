<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Signed POST Uploads

Status: delivered
Delivery set: object-storage
Plan: [Object storage plan](../../../plan/object-storage.md)
Tracker: [Object storage tracker](../../../tracker/object-storage.md)
Branch: `feat/signed-storage-uploads`
PR: `#125`

## Purpose

Let admitted clients submit private objects through exact-target signed POST
forms with provider-enforced byte, declared type and expiry conditions.

## Delivered Behavior

- Issue POST grants for one configured bucket and literal canonical key, an
  exact declared MIME type, inclusive byte range and finite effective expiry.
- Use the existing SDK to construct and sign SigV4 policies. Preserve its actual
  post-credential signing sample and bind returned expiry to the signed policy.
- Reject disabled/unknown profiles, invalid options and ineffective lifetimes.
  Never replace unsupported signing with a weaker PUT URL or wildcard policy.
- Prove actual server enforcement, reuse before expiry, expiry rejection,
  completion of an earlier transfer after expiry, and validation of a complete
  opened staging snapshot before committing those same bytes at a distinct key.

## Implementation Notes

T3.1 is `2b0802dc245271aadab6b41ce6d6968dcb004db8`; T3.2 is
`25e7d3ca3f7a78bf0a5b765043e0ae1a148cc38a`. The SDK's
`PresignPostOptions.PostPresigner` hook exposes its default signer and actual
signing time. The adapter delegates to that signer while applying an absolute
deadline, rather than adding a previously calculated duration to a later clock.
Expiry is capped by known credential expiry and rounded down to whole seconds,
matching SDK policy serialization. A non-positive effective lifetime fails;
the SDK's zero-duration default cannot extend it.

Signing is explicitly enabled with `PostPolicyMinIO` on a configured endpoint.
That named profile is a deployment assertion for the tested unversioned MinIO
release, not runtime vendor discovery. Disabled and unknown profiles fail with
`ErrUnsupported`. No application permission callback or provider registry is added.

The same pinned native fixture from Slice 2 uses source
`9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`, module
`v0.0.0-20251015172955-9e49d5e7a648`, Go 1.27.1 and observed executable SHA-256
`726b305d2753cc6b5585028b006bde0e6e7f58d7774b1308bc8b4eaa49eb5d6c`.
Only ephemeral owned buckets and loopback listeners are used. A second private
bucket proves that the signed bucket condition rejects an existing other target;
its cleanup is separately recorded. The in-flight test streams a bounded file
larger than the provider's inline threshold and verifies the accepted bytes
after EOF and expiry. Receipts contain counts, identities and hashes, not grants.

## Contracts Added or Changed

`storage` adds `UploadRequest`, `UploadGrant` and optional `UploadSigner`.
Requests require canonical keys, valid non-empty MIME metadata of at most
256 bytes, `0 <= MinBytes <= MaxBytes`, a safe positive maximum and an expiry
after signing time and at most 15 minutes away. `UploadGrant.ExpiresAt` equals
the actual encoded policy deadline and never exceeds a known credential deadline.

`storage/s3.Options.PostPolicy` selects the supported profile. `Store.SignUpload`
checks cancellation and preserves credential/context errors. Provider signing
material is bounded to 256-byte access/secret keys, a 64 KiB session token and
a 128 KiB encoded policy. Returned fields contain only required form material
and the declared content type; temporary token fields can be required, but
secret access keys and reusable signing material are never returned.

Grants are sensitive bearer capabilities. They are reusable until expiry and
cannot check current application membership, become single-use by declaration,
or revoke an already started transfer. Applications own admission, staging keys,
complete snapshot validation, distinct final destinations and durable completion.
The filesystem adapter retains its original I/O-only capability.

## Files of Interest

- `storage/upload.go`: neutral signing values, validation and optional interface.
- `storage/s3/sign.go`: SDK policy binding, profile and credential deadlines.
- `storage/s3/sign_test.go`: actual SDK policy and deterministic boundary checks.
- `storage/s3/uploads_test.go`, `inflight_test.go`: actual multipart enforcement,
  expiry, reuse and snapshot completion.
- `scripts/check-object-storage.sh`: cumulative mode 3 and required-root receipts.

## Validation

Focused ordinary checks passed:
`GOWORK=off go test -count=1 -timeout=60s ./storage ./storage/s3`.
Credential expiry arithmetic and required temporary-token fields are tested
through the SDK and deterministic boundary cases. Actual native enforcement uses
the fixture's ephemeral credentials.

The focused actual-server command passed:

```sh
HATMAX_STORAGE_MINIO="$PWD/.tmp/object-storage/tools/minio" HATMAX_STORAGE_FIXTURE_RECEIPT="$PWD/.tmp/object-storage/development-signing-fixture.txt" GOWORK=off go test -tags=integration -count=1 -timeout=360s -run '^Test(SignedUploads|SigningExpiry|SnapshotCompletion)$' ./storage/s3
```

The server rejected altered key, bucket, type, policy, signature, minimum/maximum
size, public ACL and success redirect. Admitted empty and boundary-sized files,
grant reuse, expiry, in-flight completion and distinct final snapshot bytes
passed. The focused receipt confirmed six objects before cleanup, zero afterward,
both bucket removals, server shutdown and temporary-root removal.

Final PR acceptance uses these ordered controller-recorded commands:

- `GOWORK=off scripts/check-object-storage.sh slice 3`
- `GOWORK=off make docs-check`
- `git diff --check`

All three passed at clean pushed HEAD
`33b825eb1bfa77d64423eea8a2e0c1e7ca616cbc` in controller validation
`4Q24N4WI3KUWDUNKK7Z3Q4VPRI`. Canonical PR #125 merged that commit into dev
on 2026-10-16. The native receipt confirmed eight objects cleaned to zero,
both bucket removals, owned server shutdown and temporary-root removal. Mode 3 requires
all earlier local/remote I/O roots plus TestSignedUploads, TestSigningExpiry and
TestSnapshotCompletion with race detection and confirmed fixture cleanup. It
also preserves image adapter regressions, formatting, licensing, shell syntax,
build, vet and configured ordinary/integration lint. Full mode remains unavailable.

## Risks and Follow-ups

Deployments must match the selected tested profile; arbitrary compatible vendors
are not automatically validated. Clock accuracy and credential expiry remain
operational prerequisites. A grant's expiry is not an application completion
deadline or immediate revocation mechanism. Executable guides and complete
acceptance retain their approved later owners.
