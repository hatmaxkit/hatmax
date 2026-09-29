package book

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed manifest.yaml capabilities/*.yaml archetypes/*.yaml rules/*.yaml examples/*/* release2
var defaultSource embed.FS

// LoadDefault loads the Hatmax Book embedded in this package.
func LoadDefault() (*Book, error) {
	return Load(defaultSource)
}

// LoadRelease loads one embedded Book release without changing the delivered
// default selected by existing product surfaces.
func LoadRelease(version int) (*Book, error) {
	switch version {
	case 1:
		return LoadDefault()
	case 2:
		release, err := fs.Sub(defaultSource, "release2")
		if err != nil {
			return nil, fmt.Errorf("open Book release 2: %w", err)
		}

		return Load(overlayFS{primary: release, fallback: defaultSource})
	default:
		return nil, validationError("book_release_unsupported", manifestPath, "Book release %d is not embedded", version)
	}
}

type overlayFS struct {
	primary  fs.FS
	fallback fs.FS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	file, err := o.primary.Open(name)
	if err == nil || !isNotExist(err) {
		return file, err
	}

	return o.fallback.Open(name)
}

func (o overlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := make(map[string]fs.DirEntry)

	for _, source := range []fs.FS{o.fallback, o.primary} {
		values, err := fs.ReadDir(source, name)
		if err != nil {
			if isNotExist(err) {
				continue
			}

			return nil, err
		}

		for _, entry := range values {
			entries[entry.Name()] = entry
		}
	}

	if len(entries) == 0 {
		return nil, fs.ErrNotExist
	}

	result := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}

	sort.Slice(result, func(left, right int) bool {
		return result[left].Name() < result[right].Name()
	})

	return result, nil
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
