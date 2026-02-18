package scheduler

import "time"

type Config struct {
	Enabled       bool
	Interval      time.Duration
	BatchSize     int
	Workers       int
	RetryAttempts int
	RetryBackoff  time.Duration
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
