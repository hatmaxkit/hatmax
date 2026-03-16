package ui

import (
	"context"
	"embed"
	"io/fs"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/settings"
)

// CSRFFunc extracts a CSRF token from context.
type CSRFFunc func(ctx context.Context) string

// Kit holds the UI kit configuration and dependencies.
type Kit struct {
	cfg      *config.Config
	log      log.Logger
	settings *settings.Service
	embeds   embed.FS
	overlay  fs.FS
	csrfFunc CSRFFunc
}

// Option configures the kit.
type Option func(*Kit)

// New creates a new UI kit.
func New(cfg *config.Config, log log.Logger, opts ...Option) *Kit {
	k := &Kit{
		cfg: cfg,
		log: log,
	}
	for _, opt := range opts {
		opt(k)
	}

	return k
}

// WithSettings enables settings-based form generation.
func WithSettings(svc *settings.Service) Option {
	return func(k *Kit) {
		k.settings = svc
	}
}

// WithEmbeds sets the embedded filesystem for assets.
func WithEmbeds(fs embed.FS) Option {
	return func(k *Kit) {
		k.embeds = fs
	}
}

// WithOverlay sets a filesystem overlay for asset customization.
func WithOverlay(fs fs.FS) Option {
	return func(k *Kit) {
		k.overlay = fs
	}
}

// WithCSRFFunc sets the function to extract CSRF tokens from context.
func WithCSRFFunc(fn CSRFFunc) Option {
	return func(k *Kit) {
		k.csrfFunc = fn
	}
}

// CSRFToken extracts the CSRF token from context using the configured function.
func (k *Kit) CSRFToken(ctx context.Context) string {
	if k.csrfFunc == nil {
		return ""
	}

	return k.csrfFunc(ctx)
}

// Config returns the kit's config.
func (k *Kit) Config() *config.Config {
	return k.cfg
}

// Logger returns the kit's logger.
func (k *Kit) Logger() log.Logger {
	return k.log
}

// Settings returns the kit's settings service.
func (k *Kit) Settings() *settings.Service {
	return k.settings
}
