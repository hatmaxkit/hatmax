<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Store and Resize Images

Use an `image.Store` for bytes and an `image.Processor` for variants. The
application owns metadata persistence through `image.Repository`.

## Select a store

For local files:

```go
store := local.NewStore("var/uploads", "/uploads")
```

For S3-compatible storage:

```go
store, err := s3.NewStore(ctx, s3.Options{
	Bucket:    "images",
	Region:    "us-east-1",
	Endpoint:  os.Getenv("S3_ENDPOINT"),
	AccessKey: os.Getenv("S3_ACCESS_KEY"),
	SecretKey: os.Getenv("S3_SECRET_KEY"),
	BaseURL:   "https://cdn.example.com",
})
```

## Resize and store a variant

```go
processor := stdprocessor.New()
result, err := processor.Resize(ctx, source, contentType, 800, 800)
if err != nil {
	return err
}

err = store.Put(ctx, "images/photo-medium.jpg", result.Data)
```

The processor preserves aspect ratio. PNG input remains PNG. Other decoded
formats are encoded as JPEG when a resize is required.

## Verify the result

Open the stored object through `store.Get`, then check `result.Width`,
`result.Height`, and `store.URL(path)`. Persist the corresponding `Image` and
`Variant` records through the application's repository.

See [Image Reference](../../reference/image/README.md).
