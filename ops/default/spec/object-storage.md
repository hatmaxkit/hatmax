<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Object Storage Capabilities

Date: 2026-10-16
Status: Approved
Approved: 2026-10-16
Concern: object-storage
Source ticket: [Bounded object storage and signed uploads](../ticket/reviewing/20261016081501-support-bounded-object-storage-and-signed-uploads.md)
Inspected baseline: `82defb7b51f71436ae71cbb5b2e2c980bca359b1`
Implementation status: Pending
Plan: [Delivery plan](../plan/object-storage.md)
Tracker: [Delivery tracker](../tracker/object-storage.md)
Planning authorized: 2026-10-16
Execution gate: Open; five-slice delivery authorized on 2026-10-16

## Purpose

Give Go applications bounded private object I/O, object inspection and narrowly
scoped direct-upload instructions. Ordinary documents must not require image
dimensions or an image persistence record. Applications retain authorization,
ownership, quotas, upload completion, publication and durable recovery.

Existing `image.Store`, `image/local`, `image/s3` and `image.Processor` remain
supported. Their current URL helpers provide location strings, not application
authorization. Add a cohesive `storage` package with `storage/local` and
`storage/s3` adapters. Keep signing an explicit optional capability rather than
requiring a filesystem store to become an HTTP server.

This specification defines package behavior and public value contracts, not
database entities or persistence. It does not introduce a storage service,
tenant model, background worker or generic provider registry.

## Package Contracts

The API uses caller contexts, ordinary readers and small typed options:

```go
type Object struct {
    Key         string
    SizeBytes   int64
    ContentType string
}

type WriteOptions struct {
    ContentType string
    MaxBytes    int64
}

type ReadOptions struct {
    MaxBytes int64
}

type Store interface {
    Put(context.Context, string, io.Reader, WriteOptions) (Object, error)
    Open(context.Context, string, ReadOptions) (io.ReadCloser, Object, error)
    Stat(context.Context, string) (Object, error)
    Delete(context.Context, string) error
}

type UploadRequest struct {
    Key         string
    ContentType string
    MinBytes    int64
    MaxBytes    int64
    ExpiresAt   time.Time
}

type UploadGrant struct {
    URL       string
    Fields    map[string]string
    ExpiresAt time.Time
}

type UploadSigner interface {
    SignUpload(context.Context, UploadRequest) (UploadGrant, error)
}
```

The snippets describe declarations in `storage`; implementation files supply
the standard-library imports. Returned grants specify POST form submission,
with `Fields` copied into form data and one file part. The application supplies
its own authenticated endpoint for admitting those requests before signing.
No arbitrary JSON options, request-provided provider endpoint or callback-based
permission engine belongs to these interfaces.

`Object.ContentType` is stored or provider-reported metadata. It does not prove
format safety. An ETag is not a content hash. `Stat` is an observation, not an
atomic reservation or promise that a later read returns the same object.
Callers that require verified content read one complete bounded snapshot,
validate it and commit those same bytes under another application-controlled key.

## Keys, Configuration and Errors

Use a canonical portable key contract: 1-512 ASCII bytes, non-empty slash-separated
segments, letters/digits/hyphen/underscore/dot only, and no `.` or `..` segment.
Reject absolute paths, backslashes, empty segments and malformed encodings;
do not silently normalize a rejected key into a different target. These rules
apply to the new API and do not tighten historical image-store key behavior.
Buckets, endpoints and filesystem roots are constructor-owned configuration.

Configuration errors fail construction or explicit initialization. Production
remote endpoints use verified TLS. A clearly named insecure option may support
isolated loopback/container fixtures; it must never disable TLS verification
implicitly. Support the existing AWS SDK credential mechanisms, including
expiring credentials, without exposing credentials through the neutral API.
No public base URL or public-read bucket is needed for private storage.

Classify errors with `errors.Is`: `ErrInvalid` for invalid options/keys,
`ErrNotFound` for confirmed object absence, `ErrTooLarge` for an enforced byte
bound, `ErrUnsupported` for an unavailable capability and `ErrIndeterminate`
when a write/delete outcome cannot be established. Preserve context cancellation
and deadlines through wrapping. Authentication, permission, malformed provider
responses and transport failures must not be collapsed into absence.

`Delete` succeeds when the object is already confirmed absent. A lost response
or provider timeout is not proof of deletion. Do not retry mutations silently;
callers own retries and reconciliation. Read-only bounded retries must respect
the operation context and a finite configured SDK attempt count.

## Bounded I/O and Resource Ownership

Reads and writes require an explicit positive `MaxBytes` within safe int64
arithmetic. Reject invalid bounds before creating files or sending network
requests. Empty objects are permitted by the generic store; application policy
may require a non-empty upload. Reject an oversized object on observed metadata
when possible and enforce the actual streamed-byte limit independently. Size
metadata cannot make a longer body acceptable.

`Open` returns metadata for that opened response and an owned reader. The caller
closes it. A read beyond the bound returns a classified size error rather than
silently presenting truncated success. Only reading to successful EOF establishes
a complete snapshot. The adapter closes a response it rejects before returning.

`Put` does not take ownership of the caller's input reader. Validate a bounded
input before committing a local replacement or issuing a remote request that
needs a replayable body. Use bounded-memory streaming and an owned private spool
when necessary; do not allocate the caller's entire maximum in memory. Spool
files have restricted permissions, a configured finite capacity and cleanup on
success and failure. Constructors do not start workers or create global queues.
Document per-operation memory/disk bounds and concurrent-call consequences.

Successful writes expose complete objects. A failure that may have committed an
object is explicitly indeterminate; the adapter cannot guarantee remote rollback.
The caller chooses immutable destination keys when overwrite would be unsafe.
Replacing an existing key is not a compare-and-swap or an exactly-once operation.
No automatic application availability or quota transition follows a store write.

Check cancellation between reads and phases and preserve request deadlines.
Do not promise that a context can interrupt an arbitrary blocking `io.Reader`
owned by the caller. No detached goroutine may continue I/O after the operation
returns merely to emulate that promise.

## Local Adapter

Confine all operations to a caller-controlled root using directory-handle
operations, including under symlink replacement races. A preflight path check
followed by an unrestricted open is insufficient. Temporary writes remain under
the owned root and replace the final target only after complete bounded input
and successful close. A failed write must not leave a readable partial final
object or corrupt the previous complete object.

Local metadata must remain meaningful across process restart; filename suffixes
alone are not proof of content type. The implementation may derive bounded
content metadata from the object rather than invent an application database.
Record that local content-type derivation and remote supplied metadata have
different provenance. `Stat`, `Open` and delete never escape the root.
The local adapter does not implement signing or serve files publicly.

## S3 Adapter and Signed POST

Reuse the already adopted AWS SDK for Go v2. Isolate SDK types and wire policy
construction in the S3 adapter; neutral contracts expose no SDK dependency.
Put/Open/Stat/Delete operate in the configured private bucket and preserve literal
admitted keys. Validate returned object size/type metadata; malformed metadata
is an error. Permission failures must not masquerade as `ErrNotFound`.

Signing uses the SDK's supported presigned POST mechanism and a SigV4 policy
containing exact bucket, exact key, exact declared content type, an explicit
content-length range and expiry. Wildcard key prefixes, empty-prefix conditions,
public ACLs and caller-controlled success redirects are not supported. A bare
presigned PUT without equivalent provider-enforced size limits is insufficient.

Require `0 <= MinBytes <= MaxBytes`, a positive maximum, a valid MIME type of at
most 256 bytes and an expiry after the trusted signing time and at most 15 minutes
away. Signing must not outlive known credential expiry. Expose the effective
expiry and never silently extend a caller deadline or fall back to the SDK's
default lifetime. Use one trusted signing-time sample shared by policy and
returned metadata; test clock-boundary behavior deterministically.

Return only provider-required form fields for that upload. Secret access keys
and reusable signing material never reach the browser. Credential identifiers
and a required temporary session-token field may be present in the signed form;
they are part of a sensitive limited grant and must not be logged or persisted
by examples. Grant contents cannot be used as general SDK credentials.

A grant is a bearer capability and may be reused until its expiry. It cannot
check application membership at the provider, become single-use by declaration,
or revoke an already started transfer. The application must use disposable
staging keys and its own current-authority completion checks. The core signer
does not issue downloads, deletes or write access to another object.

Support only providers that enforce the required POST policy. An unsupported
endpoint must fail capability validation explicitly, never downgrade to weaker
signing. Document the supported initial unversioned-bucket profile and the
different deletion semantics of versioned buckets; no claim of removing all
historical versions is made by key deletion.

## Compatibility, Documentation and Validation

Preserve existing exported image-store constructors and interfaces. Internal
shared implementation is permitted only with tests proving existing key, URL,
image-limit and error behavior; do not force all image callers to migrate.
Image processing remains image-owned. New neutral storage documentation must
not describe object metadata as an image or require image records for documents.

Deliver a local object tutorial, guides for private S3/signed forms and failure
reconciliation, exact API/configuration reference and explanation of metadata,
bearer scope and completion. Examples use existing Go/shell tooling. Signing
examples submit the actual form and inspect actual accepted bytes without
printing grant fields. No external production account is needed for acceptance.

Use real filesystem operations and an isolated MinIO-compatible S3 server built
with Go from the exact source version pinned by the independent plan. Record
the built binary digest before integration validation. This avoids depending on
an unavailable published image and adds no language runtime. Existing container
infrastructure may remain in use for other repository tests. Fixture startup,
readiness, operation and teardown
have finite deadlines and exact ownership. Missing required infrastructure is
a visible validation blocker, not a skipped green integration result.
This establishes that fixture's tested compatibility, not every provider's.

| ID | Observable acceptance |
| --- | --- |
| OS01 | A document is written, inspected, read and deleted on local and supported remote adapters without image dimensions or public URLs |
| OS02 | Invalid keys/options fail before I/O; traversal, symlink races and adversarial nested keys cannot escape the configured local root |
| OS03 | Actual read/write limits, malformed metadata, cancellation and owned reader/spool cleanup remain correct on success and failure |
| OS04 | Confirmed absence is distinguishable from permissions, transport failure and indeterminate mutation outcomes; retries do not silently duplicate writes |
| OS05 | The actual remote server accepts a signed POST only within its exact key/type/size/expiry conditions; altered key, type, policy/signature, under/oversize or expired grants are rejected |
| OS06 | Grant metadata matches effective signing/credential deadlines; documented reuse and in-flight boundaries are tested without claiming one-use or immediate revocation |
| OS07 | A single opened snapshot can be validated and written to a distinct final key without a later staging overwrite changing those bytes |
| OS08 | Existing image/local/S3/processor APIs and examples preserve their documented behavior |
| OS09 | Four documentation quadrants and executable examples match public APIs and prove actual local and signed remote workflows |
| OS10 | Exact-candidate package/integration/documentation/full-repository checks and owned cleanup pass; unsupported provider behavior cannot be reported as accepted |

## Approval State and References

The maintainer approved this core scope and authorized its independent five-slice
Looper execution on 2026-10-16. The plan/tracker bind exact identities, fixture
source/build, commands and restricted closure. Authority includes Rebase +
Fast-forward into dev and the complete gate; it neither declares delivery nor
authorizes a version/release.

Source references: `image/store.go`, `image/local/store.go`, `image/s3/store.go`,
and the pinned AWS SDK's `PresignPostObject`/`PresignPostOptions.Conditions`.
Provider semantics: [S3 POST policy](https://docs.aws.amazon.com/AmazonS3/latest/developerguide/sigv4-HTTPPOSTConstructPolicy.html),
[presigned request lifetime](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html)
and [MinIO POST policy implementation](https://github.com/minio/minio/blob/master/cmd/postpolicyform.go).
