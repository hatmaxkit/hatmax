package scheduler

import (
	"time"

	"hatmax.adrianpk.com/config"
)

type Config struct {
	Enabled       bool
	Interval      time.Duration
	BatchSize     int
	Workers       int
	RetryAttempts int
	RetryBackoff  time.Duration
}

// ConfigFromRoot maps the shared root config into scheduler config.
func ConfigFromRoot(cfg *config.Config) Config {
	if cfg == nil {
		return Config{}.WithDefaults()
	}

	return Config{
		Enabled:       cfg.Scheduler.Enabled,
		Interval:      cfg.Scheduler.IntervalDuration(),
		BatchSize:     cfg.Scheduler.BatchSize,
		Workers:       cfg.Scheduler.Workers,
		RetryAttempts: cfg.Scheduler.RetryAttempts,
		RetryBackoff:  cfg.Scheduler.RetryBackoffDuration(),
	}.WithDefaults()
}

func (c Config) WithDefaults() Config {
	if c.Interval <= 0 {
		c.Interval = time.Minute
	}

	if c.BatchSize <= 0 {
		c.BatchSize = 20
	}

	if c.Workers <= 0 {
		c.Workers = 1
	}

	if c.RetryAttempts <= 0 {
		c.RetryAttempts = 3
	}

	if c.RetryBackoff <= 0 {
		c.RetryBackoff = time.Minute
	}

	return c
}

const (
	SettingEnabled  = "scheduler.enabled"
	SettingInterval = "scheduler.interval_seconds"
	SettingPaused   = "scheduler.paused"
)
