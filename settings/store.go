package settings

import "context"

// Store defines the persistence interface for settings.
type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	All(ctx context.Context) ([]Value, error)
	Delete(ctx context.Context, key string) error
}
