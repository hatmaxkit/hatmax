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

`local.NewStore(basePath, baseURL)` stores those strings. `Put` creates
missing parent directories with mode `0755` and writes the file under
`basePath`. `Get` opens that file. `Delete` removes that file and does not
remove directories. `URL` returns `baseURL + "/" + path`. `BasePath` returns
the filesystem root.

## S3 store

`s3.NewStore` requires `Bucket` and `BaseURL`. A missing value returns `s3:
bucket is required` or `s3: base_url is required`. `Region`, `AccessKey`, and
`SecretKey` configure the AWS client. A non-empty `Endpoint` sets that base
endpoint and path-style addressing. `BaseURL` is stored without a trailing
slash.

`Put` reads the body, sets `Content-Type` from `http.DetectContentType`, and
stores the object at the path. `Get` returns the object body. `Delete`
deletes that key. `URL` returns `baseURL + "/" + path`.

## Standard processor

`stdprocessor.New` uses JPEG quality 85. `NewWithQuality` clamps a value
below 1 to 1 and a value above 100 to 100.

`Resize` decodes the reader. When both calculated dimensions are at least the
original dimensions, it returns the original bytes, the original size, and
the supplied content type. Otherwise it scales with Catmull-Rom, keeps PNG as
PNG, and encodes JPEG and every other decoded format as JPEG at the
configured quality. The context argument is not used.

The scale keeps the aspect ratio inside `maxWidth` and `maxHeight`. An image
already inside both limits keeps its dimensions.

`GetDimensions` returns the decoded width and height. It ignores
`contentType` and the context. `DetectContentType` returns
`http.DetectContentType`.
