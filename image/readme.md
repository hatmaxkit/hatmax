<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# image

Image storage and processing with variants.

## Usage

```go
// Create store (local or S3)
store := local.NewStore("/var/uploads", "https://cdn.example.com")
store, err := s3.NewStore(ctx, s3.Options{
    Bucket:  "my-bucket",
    Region:  "us-east-1",
    BaseURL: "https://cdn.example.com",
})

// Store image
err := store.Put(ctx, "images/photo.jpg", reader)

// Get image
reader, err := store.Get(ctx, "images/photo.jpg")

// Get URL
url := store.URL("images/photo.jpg")

// Process one variant
processor := stdprocessor.New()
resized, err := processor.Resize(ctx, original, "image/jpeg", 800, 800)
if err != nil { ... }
err = store.Put(ctx, "images/photo-medium.jpg", resized.Data)
```

## API

```go
type Store interface {
    Put(ctx context.Context, path string, data io.Reader) error
    Get(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    URL(path string) string
}
```

Implementations: `local/`, `s3/`. Processor: `stdprocessor/`.

`DefaultVariantSpecs` returns the standard large, medium, and thumbnail
dimensions. The application chooses paths and persists `Image` and `Variant`
metadata through its `Repository` implementation.
