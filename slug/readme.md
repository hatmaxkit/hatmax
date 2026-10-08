<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# slug

ASCII URL slugs with accent-mark removal. This complete example prints two slugs:

## Usage

```go
package main

import (
	"fmt"

	"github.com/google/uuid"
	"hatmax.adrianpk.com/slug"
)

func main() {
	id := uuid.MustParse("3995fd11-1234-5678-9abc-def012345678")
	fmt.Println(slug.Generate("Hello World!", id))
	fmt.Println(slug.Normalize("Café & Résumé", 50))
}
```

Output: `hello-world-3995fd11` then `cafe-resume`. Normalization removes Unicode
combining marks, retains only ASCII letters and digits, and joins separators
with hyphens. It does not transliterate every writing system. Truncation prefers
a hyphen boundary; a single long word is cut at the byte limit. `Generate` adds
eight UUID characters after at most 50 normalized characters, so it can return
59 characters. That prefix does not guarantee uniqueness. See the
[Slug Reference](../docs/reference/slug/README.md).
