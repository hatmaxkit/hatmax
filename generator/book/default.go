package book

import "embed"

//go:embed manifest.yaml capabilities/*.yaml archetypes/*.yaml rules/*.yaml examples/*/*
var defaultSource embed.FS

// LoadDefault loads the Hatmax Book embedded in this package.
func LoadDefault() (*Book, error) {
	return Load(defaultSource)
}
