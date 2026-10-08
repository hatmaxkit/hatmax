<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Store and Resize Images

Use an `image.Store` for bytes and an `image.Processor` for variants. The
application owns metadata persistence through `image.Repository`.

## Select a store

Inside the upload service, with a caller-owned reader and context, select one
store. For local files, the application must control this directory and its
ancestors:

```go
store := local.NewStore("var/uploads", "/uploads")
```

Alternatively, for S3-compatible storage in a function returning an error:

```go
store, err := s3.NewStore(ctx, s3.Options{
	Bucket:    "images",
	Region:    "us-east-1",
	Endpoint:  os.Getenv("S3_ENDPOINT"),
	AccessKey: os.Getenv("S3_ACCESS_KEY"),
	SecretKey: os.Getenv("S3_SECRET_KEY"),
	BaseURL:   "https://cdn.example.com",
})
if err != nil {
	return err
}
```

## Resize and store a variant

```go
processor := stdprocessor.New()
result, err := processor.Resize(ctx, source, contentType, 800, 800)
if err != nil {
	return err
}

path := "images/photo-medium.jpg"
if result.ContentType == "image/png" {
	path = "images/photo-medium.png"
}
if err := store.Put(ctx, path, result.Data); err != nil {
	return err
}
```

The processor preserves aspect ratio. PNG input remains PNG. Other decoded
formats are encoded as JPEG when a resize is required.
An input already inside the bounds is returned with its original bytes and
caller-supplied content type. Validate that type and choose a matching extension;
the fragment assumes JPEG or PNG input. Resizing rejects encoded input over
20 MiB, images over 25 million pixels and non-positive bounds. The application
owns upload concurrency, deadlines and input-reader cleanup. Context checks
occur between reads and CPU phases; they cannot interrupt a blocked reader or
codec operation. Local `Put` does not apply these limits itself.

## Verify the result

Open the stored object through `store.Get`, close that reader after inspecting
it, and confirm its dimensions and `store.URL(path)`. The complete local
[package example](../../../image/readme.md#usage) does this readback. Delete
test objects with `store.Delete`; on copy errors, clean up any partial local
file. Persist the corresponding `Image` and `Variant` records through the
application's repository and compensate for partial workflow failure.

S3 construction and URL generation do not prove upload credentials or bucket
access. Use a controlled account for provider acceptance; local files and a
fixture endpoint establish only their exercised adapter behavior.

See [Image Reference](../../reference/image/README.md).
