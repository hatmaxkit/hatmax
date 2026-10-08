<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# image

Image storage and processing with variants. The application owns image metadata,
upload admission and authorization.

## Usage

This complete function accepts a caller-owned PNG reader and an
application-controlled upload directory. Call it from the service that owns the
upload; the stored variant remains until that service deletes it:

```go
package example

import (
	"context"
	"fmt"
	"io"

	"hatmax.adrianpk.com/image/local"
	"hatmax.adrianpk.com/image/stdprocessor"
)

func savePNG(ctx context.Context, uploadRoot string, source io.Reader) (string, error) {
	store := local.NewStore(uploadRoot, "/uploads")
	processor := stdprocessor.New()
	result, err := processor.Resize(ctx, source, "image/png", 800, 800)
	if err != nil {
		return "", err
	}
	path := "images/photo-medium.png"
	if err := store.Put(ctx, path, result.Data); err != nil {
		return "", err
	}
	stored, err := store.Get(ctx, path)
	if err != nil {
		return "", err
	}
	defer stored.Close()
	width, height, err := processor.GetDimensions(ctx, stored, result.ContentType)
	if err != nil {
		return "", err
	}
	if width != result.Width || height != result.Height {
		return "", fmt.Errorf("stored dimensions differ from the processed variant")
	}
	return store.URL(path), nil
}
```

For a valid PNG, the returned URL is `/uploads/images/photo-medium.png` and
dimensions fit within 800 by 800 without enlargement. `Get` readers must be
closed by the caller. `URL` only builds a string; the application must serve it.
S3 selection and required credentials are shown in the
[storage how-to](../docs/how-to/store-images/README.md).

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

Standard resizing and S3 uploads accept at most 20 MiB of encoded input.
The standard processor checks dimensions before full decoding and rejects
images above 25 million pixels. Dimension inspection uses the same pixel
limit without reading the full body. Resize bounds must be positive.
Both adapters check cancellation between reads; resizing also checks between
CPU phases. They do not close caller-owned readers or preempt blocked reads
and codec/scaling computation. See the
[image reference](../docs/reference/image/README.md#standard-processor)
for the fixed limits and cancellation contract.

Local storage accepts relative object paths and confines filesystem operations
to its configured root. Traversal and escaping symlinks cannot read, overwrite,
or remove files outside that root. The application must control the root and
its ancestors. See the [local store reference](../docs/reference/image/README.md#local-store)
for path validation, symlink semantics, and platform guarantees.
Unlike the processor and S3 adapter, local storage does not inspect the context
or impose a byte limit. It copies input directly; callers own bounded reads,
cancellation of their readers and removal of partial files after copy errors.

`DefaultVariantSpecs` returns the standard large, medium, and thumbnail
dimensions. The application chooses paths and persists `Image` and `Variant`
metadata through its `Repository` implementation.
