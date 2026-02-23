package settings

import (
	"context"
	"strconv"
)

// Service provides type-safe access to settings.
type Service struct {
	registry *Registry
	store    Store
}

// NewService creates a new settings service.
func NewService(registry *Registry, store Store) *Service {
	return &Service{registry: registry, store: store}
}

// GetString returns the setting value or its default.
func (s *Service) GetString(ctx context.Context, key string) (string, error) {
	raw, err := s.store.Get(ctx, key)
	if err != nil || raw == "" {
		return s.defaultFor(key), nil
	}

	return raw, nil
}

// GetInt returns the setting value parsed as int, or its default.
func (s *Service) GetInt(ctx context.Context, key string) (int, error) {
	raw, err := s.store.Get(ctx, key)
	if err != nil || raw == "" {
		return s.defaultInt(key)
	}

	return strconv.Atoi(raw)
}

// GetBool returns the setting value parsed as bool, or its default.
func (s *Service) GetBool(ctx context.Context, key string) (bool, error) {
	raw, err := s.store.Get(ctx, key)
	if err != nil || raw == "" {
		return s.defaultBool(key)
	}

	return strconv.ParseBool(raw)
}

// Set validates and persists a setting value.
func (s *Service) Set(ctx context.Context, key, value string) error {
	if schema, ok := s.registry.Get(key); ok {
		err := schema.Validate(value)
		if err != nil {
			return err
		}
	}

	return s.store.Set(ctx, key, value)
}

// Delete removes a setting.
func (s *Service) Delete(ctx context.Context, key string) error {
	return s.store.Delete(ctx, key)
}

// All returns all stored settings.
func (s *Service) All(ctx context.Context) ([]Value, error) {
	return s.store.All(ctx)
}

func (s *Service) defaultFor(key string) string {
	if schema, ok := s.registry.Get(key); ok {
		return schema.Default
	}

	return ""
}

func (s *Service) defaultInt(key string) (int, error) {
	def := s.defaultFor(key)
	if def == "" {
		return 0, nil
	}

	return strconv.Atoi(def)
}

func (s *Service) defaultBool(key string) (bool, error) {
	def := s.defaultFor(key)
	if def == "" {
		return false, nil
	}

	return strconv.ParseBool(def)
}
