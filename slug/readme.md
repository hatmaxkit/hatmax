# slug

URL-friendly slug generation with Unicode support.

## Usage

```go
// Generate slug from text + UUID
id := uuid.MustParse("3995fd11-1234-5678-9abc-def012345678")
s := slug.Generate("Hello World!", id)
// -> "hello-world-3995fd11"

// Normalize only (no UUID suffix)
s := slug.Normalize("Café & Résumé", 50)
// -> "cafe-resume"
```

Handles accented characters, special chars, and truncates at word boundaries.
