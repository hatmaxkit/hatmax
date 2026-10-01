<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Image

`image` defines stored images, variants, a storage boundary, a persistence
boundary, and a processor boundary. `image/local`, `image/s3`, and
`image/stdprocessor` implement storage and processing. The implementation
note is [image/readme.md](../../../image/readme.md).

## Records

`Image` fields are `ID`, `Filename`, `ContentType`, `SizeBytes`, `Width`,
`Height`, `StoragePath`, `Metadata`, `CreatedAt`, and `UpdatedAt`.

`Variant` fields are `ID`, `OriginalID`, `Type`, `Width`, `Height`,
`StoragePath`, `SizeBytes`, and `CreatedAt`.

`VariantType` values are `original`, `large`, `medium`, and `thumbnail`.
`ValidVariantTypes` returns those four values.

`DefaultVariantSpecs` returns large at 1200 by 1200, medium at 800 by 800,
and thumbnail at 300 by 300. It does not include `original`.

## Boundaries

`Store` is `Put`, `Get`, `Delete`, and `URL`. `Put` stores the reader at the
supplied path. `Get` returns a reader. `Delete` removes that path. `URL`
returns a servable URL.

`Repository` is `Create`, `Get`, and `Delete` for an `Image`, plus
`CreateVariant`, `GetVariants`, `GetVariant`, and `DeleteVariants`. This
package does not supply an implementation.

`Processor` is `Resize`, `GetDimensions`, and `DetectContentType`.
`ProcessedImage` has `Data`, `Width`, `Height`, `SizeBytes`, and
`ContentType`.

## Local store

`local.NewStore(basePath, baseURL)` stores those strings without opening the
filesystem. The configured root and its ancestors must be controlled by the
application. `Put` creates a missing root and parent directories with mode
`0755`. `Get` and `Delete` do not create the root.

All three operations require a local, non-root object path. Empty paths,
absolute paths, paths that escape lexically, and paths that normalize to `.`
return an `os.PathError` wrapping `fs.ErrInvalid`. Nested paths remain valid;
each accepted path is normalized with `filepath.Clean`.

Each operation opens and closes an `os.Root` handle. Parent creation, file
access, and deletion stay beneath that handle, including when a path component
is replaced by an escaping symbolic link during the operation. Relative
symlinks that resolve within the root are allowed. Absolute symlinks are not
followed, even when their targets are inside the root.

`Get` returns an independent file handle; the caller must close it. `Delete`
removes the named entry without recursive deletion. A final symlink is
unlinked, not followed, so removing it does not remove its target.

Confinement inherits the [platform guarantees of Go's os.Root](https://pkg.go.dev/os#Root).
It does not sandbox hard links, privileged mounts, or device files.
`GOOS=js` does not provide protection against symlink replacement races.

`URL` remains `baseURL + "/" + path`, without object-path validation or
filesystem access. `BasePath` returns the configured string unchanged.

## S3 store

`s3.NewStore` requires `Bucket` and `BaseURL`. A missing value returns `s3:
bucket is required` or `s3: base_url is required`. `Region`, `AccessKey`, and
`SecretKey` configure the AWS client. A non-empty `Endpoint` sets that base
endpoint and path-style addressing. `BaseURL` is stored without a trailing
slash.

`Put` accepts at most 20 MiB (20,971,520 bytes), reading at most one additional
byte to detect overflow. Oversized input and read failures return an error
before any upload. It checks context cancellation before and after each read,
sets `Content-Type` from `http.DetectContentType`, and uploads directly from
the byte buffer without an intermediate string copy. Empty objects remain
valid. The AWS request also uses the caller's context.
`Get` returns the object body. `Delete`
deletes that key. `URL` returns `baseURL + "/" + path`.

## Standard processor

`stdprocessor.New` uses JPEG quality 85. `NewWithQuality` clamps a value
below 1 to 1 and a value above 100 to 100.

`Resize` accepts at most 20 MiB of encoded input, with one overflow-detection
byte. It calls `image.DecodeConfig` before full decoding and rejects
non-positive dimensions or more than 25,000,000 pixels. Pixel accounting uses
division to avoid multiplication overflow. Resize bounds must be positive.
These are fixed limits; the existing constructors and interfaces are unchanged.

When both calculated dimensions are at least the
original dimensions, it returns the original bytes, the original size, and
the supplied content type. Otherwise it scales with Catmull-Rom, keeps PNG as
PNG, and encodes JPEG and every other decoded format as JPEG at the
configured quality. Cancellation is checked between input reads and before
or after decode, scale, and encode phases. A canceled operation does not return
a successful result. The standard codecs and Catmull-Rom scaling do not offer
mid-computation cancellation.

The scale keeps the aspect ratio inside `maxWidth` and `maxHeight`. An image
already inside both limits keeps its dimensions.

`GetDimensions` reads only image configuration and applies the same pixel
limit. Configuration reads have the 20 MiB budget and context checks; this
operation does not drain or validate the remaining encoded body. It ignores
`contentType`. `DetectContentType` returns
`http.DetectContentType`.

Neither adapter takes ownership of the input reader. A blocked arbitrary
`io.Reader.Read` cannot be interrupted by context alone; callers must supply
a reader with appropriate I/O deadlines or cancellation when necessary.
Encoded-byte and pixel limits bound individual inputs, not total process
memory or request concurrency. Applications still own upload admission and
concurrent processing limits. Registered codecs must report consistent
configuration and decoded dimensions.
