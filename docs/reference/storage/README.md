<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Object Storage Reference

Packages `storage`, `storage/local` and `storage/s3` provide bounded private
object I/O. They add no image dimensions, public URLs, persistence records,
authorization callbacks, workers or provider registry. Exported Go declarations
are authoritative. See [the tutorial](../../tutorials/private-objects/README.md)
for a learning path and [private configuration](../../how-to/store-private-objects/README.md)
for setup.

## Neutral values and methods

| Value | Exact fields and contract |
| --- | --- |
| `Object` | `Key string`, `SizeBytes int64`, `ContentType string`; observed metadata, not validation or a reservation. |
| `WriteOptions` | `ContentType string`, `MaxBytes int64`; type optional, positive maximum below `math.MaxInt64`. |
| `ReadOptions` | `MaxBytes int64`; positive and below `math.MaxInt64`, enforced on metadata and actual stream. |
| `UploadRequest` | `Key string`, `ContentType string`, `MinBytes int64`, `MaxBytes int64`, `ExpiresAt time.Time`. |
| `UploadGrant` | `URL string`, `Fields map[string]string`, `ExpiresAt time.Time`; sensitive multipart POST material and actual policy deadline. |

`Store` exposes:

- `Put(context.Context, string, io.Reader, WriteOptions) (Object, error)`:
  admit and fully spool bounded input before committing a complete object.
  Never close the caller's input. Failed uncertain mutations may have committed.
- `Open(context.Context, string, ReadOptions) (io.ReadCloser, Object, error)`:
  return one owned reader with metadata for that opened response. Only successful
  EOF is complete; actual overflow returns `ErrTooLarge`, not truncated success.
- `Stat(context.Context, string) (Object, error)`: observe metadata without
  reserving a later snapshot.
- `Delete(context.Context, string) error`: confirmed absence succeeds. No silent
  mutation retry, historical-version purge, compare-and-swap or exactly-once claim.

`UploadSigner` is optional and independent of Store:
`SignUpload(context.Context, UploadRequest) (UploadGrant, error)`.
The local store does not implement it. `UploadRequest.Validate(time.Time) error`,
`ReadOptions.Validate() error`, `WriteOptions.Validate() error` and
`ValidateKey(string) error` validate before I/O. Signing validates again at the
actual SDK signing sample after credential retrieval.

## Keys, types and upload bounds

Keys contain 1-512 ASCII bytes in nonempty slash-separated segments. Only letters,
digits, dot, hyphen and underscore are admitted; `.` and `..` segments are rejected.
Absolute paths, trailing/repeated slashes, backslashes, percent encodings and
non-ASCII are rejected without normalization. These rules apply to the new API;
historical image-store key behavior is preserved.

Nonempty MIME metadata must parse as a media type, include a slash, use at most
256 bytes and contain no CR, LF or NUL. Put permits empty metadata. Signed uploads
require a nonempty type and `0 <= MinBytes <= MaxBytes`, with positive safe maximum.
Requested expiry must be after trusted signing time and at most 15 minutes away.
The actual policy is capped by known credential expiry and rounded down to whole
seconds; a non-positive effective lifetime fails. Returned expiry equals that
policy deadline, with no silent SDK default extension.

The SDK constructs and signs exact bucket/key/type equality conditions, inclusive
file-byte range and finite expiry. No wildcard prefix, public ACL, caller success
redirect or weaker PUT grant is supported. Copy all fields unchanged and append
one file part last. Credential identifiers and required temporary-token fields
may appear; secret access keys and reusable signing material do not. Signing
material is bounded to 256-byte access and secret keys, a 64 KiB session token
and 128 KiB encoded policy. Grants remain sensitive and must not be logged or
persisted. Reuse and in-flight boundaries are described in
[Bearer and Snapshot Ownership](../../explanation/private-object-ownership/README.md).

## Local constructor and resources

`local.NewStore(directory string, options local.Options) (*local.Store, error)`
opens an existing application-owned root. `local.Options.MaxSpoolBytes int64` is
required and follows the safe positive bound. Each Put's maximum must fit this
capacity. The store owns an `os.Root`; `Close() error` releases it without closing
previously returned readers. Callers control initial ancestors, contents, hard
links and mounts. Directory-handle operations confine symlink replacement races.

Private spool/object files use mode 0600 and created directories mode 0700.
Each write uses a fixed 32 KiB copy buffer plus bounded disk input before atomic
replacement. N concurrent writes use up to N times the configured capacity plus
filesystem overhead. Spools are removed on success and failure. Atomic replacement
does not promise crash durability. Content type derives from at most 512 bytes
with `http.DetectContentType` on each observation, including after reopening;
WriteOptions.ContentType is validated but not persisted.

## S3 constructor and options

`s3.NewStore(context.Context, s3.Options) (*s3.Store, error)` validates configuration,
opens an existing spool directory and starts no workers. Close readers and finish
operations before `Close() error`, which releases the spool handle and owned idle
HTTP connections. Buckets and spool directories are not created by construction.

| `s3.Options` field | Contract |
| --- | --- |
| `Bucket string` | Required lowercase DNS-style name of 3-63 bytes; no IP literal, repeated dots or leading/trailing punctuation. |
| `Region string` | Required nonblank region. |
| `Endpoint string` | Optional for I/O using normal SDK resolution; explicit endpoint required by the POST profile. Custom endpoints use path addressing. |
| `Credentials aws.CredentialsProvider` | Optional; otherwise SDK default chain, including renewable credentials. |
| `SpoolDirectory string` | Required existing application-controlled directory. |
| `MaxSpoolBytes int64` | Required safe positive per-write capacity; each write maximum must fit it. |
| `ReadAttempts int` | Zero defaults to 1; explicit range 1-5, read-only attempts under one operation context. |
| `OperationTimeout time.Duration` | Zero defaults to 30 seconds; positive maximum 120 seconds, including returned reader lifetime. |
| `AllowInsecureLoopback bool` | HTTP only when explicitly enabled and endpoint hostname is a literal loopback address. HTTPS always verifies certificates. |
| `PostPolicy s3.PostPolicyProfile` | Disabled by default; `PostPolicyMinIO` is the explicitly tested unversioned profile and requires an endpoint. Unknown profiles fail `ErrUnsupported`. |

Endpoints allow no userinfo, query, fragment or non-root path. No redirects are
followed. Remote Put spools complete input privately before one mutation attempt,
using a fixed 32 KiB copy buffer; the capacity and concurrent-call disk bounds
match the local description. Open validates provider metadata and independently
limits streamed bytes. Returned provider content type remains metadata; it is
not filename-derived, a hash or format verification. Empty Put type may be
provider-inferred; Open/Stat validate returned metadata.

The initial signing profile is MinIO `RELEASE.2025-10-15T17-29-55Z`, source
`9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`, Go module
`github.com/minio/minio v0.0.0-20251015172955-9e49d5e7a648`, built with Go 1.27.1.
Selecting it asserts deployment compatibility; it does not discover a vendor.
Tests establish that identified server's behavior, not arbitrary S3 vendors.
Disabled signing returns `ErrUnsupported`; there is no fallback. Delete removes
the current key in this unversioned profile, not all historical versions.

## Errors and cancellation

Use `errors.Is` for `ErrInvalid`, `ErrNotFound`, `ErrTooLarge`, `ErrUnsupported`
and `ErrIndeterminate`. Open/Stat absence maps only confirmed local absence or
remote 404 `NoSuchKey`/`NotFound`; authentication, denial, malformed metadata and
transport failures retain their causes. Uncertain mutation transport/server
outcomes are indeterminate; a failed response is not proof of rollback.
Context cancellation/deadlines survive wrapping and can accompany uncertainty.

Cancellation is checked between input reads and processing phases. It cannot
interrupt an arbitrary already-blocked caller Read; supply I/O deadlines or
cancellable readers. No detached worker reads after Put returns. The application
owns total concurrency, quotas, retries and durable recovery. Follow the
[reconciliation procedure](../../how-to/reconcile-object-storage/README.md).

## Image compatibility

Existing `image.Store` Put/Get/Delete/URL, `image/local.NewStore(basePath, baseURL)`,
`image/s3.NewStore(ctx, image/s3.Options)` and `image.Processor` remain supported.
Their accepted keys, constructors, URL strings, errors and image limits are
unchanged. URL helpers construct locations, not application authorization.
General documents need no image record. See [Image Reference](../image/README.md).
