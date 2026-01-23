# image

Image storage and processing with variants.

## Usage

```go
// Create store (local or S3)
store := local.NewStore("/var/uploads", "https://cdn.example.com")
store := s3.NewStore(s3Client, bucket, "https://cdn.example.com")

// Store image
err := store.Put(ctx, "images/photo.jpg", reader)

// Get image
reader, err := store.Get(ctx, "images/photo.jpg")

// Get URL
url := store.URL("images/photo.jpg")

// Process variants
processor := stdprocessor.New()
variants := []image.Variant{image.Large, image.Medium, image.Thumbnail}
for _, v := range variants {
    resized, _ := processor.Resize(original, v.Width, v.Height)
    store.Put(ctx, v.Path(basePath), resized)
}
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
