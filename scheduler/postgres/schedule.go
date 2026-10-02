// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

// Stored schedules describe the existing schedule types, not executable commands.
func parseSchedule(spec, zone string) (scheduler.Schedule, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}

	var value struct {
		Type   string `json:"type"`
		Every  string `json:"every"`
		Hour   int    `json:"hour"`
		Minute int    `json:"minute"`
		Day    int    `json:"day"`
	}

	decoder := json.NewDecoder(strings.NewReader(spec))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&value)
	if err != nil {
		return nil, fmt.Errorf("scheduler: invalid schedule: %w", err)
	}

	err = decoder.Decode(new(any))
	if err != io.EOF {
		return nil, fmt.Errorf("scheduler: schedule must contain one JSON object")
	}

	if value.Type == "interval" {
		every, err := time.ParseDuration(value.Every)
		if err != nil || every <= 0 || value.Hour != 0 || value.Minute != 0 || value.Day != 0 {
			return nil, fmt.Errorf("scheduler: interval requires a positive duration and no calendar fields")
		}

		return scheduler.Interval{Every: every}, nil
	}

	if value.Hour < 0 || value.Hour > 23 || value.Minute < 0 || value.Minute > 59 || value.Every != "" {
		return nil, fmt.Errorf("scheduler: invalid calendar schedule")
	}

	if zone == "" {
		zone = "UTC"
	}

	location, err := time.LoadLocation(zone)
	if err != nil {
		return nil, fmt.Errorf("scheduler: invalid schedule timezone: %w", err)
	}

	switch value.Type {
	case "daily":
		if value.Day != 0 {
			return nil, fmt.Errorf("scheduler: daily schedule cannot specify a weekday")
		}

		return scheduler.Daily{Hour: value.Hour, Minute: value.Minute, TZ: location}, nil
	case "weekly":
		if value.Day < 0 || value.Day > 6 {
			return nil, fmt.Errorf("scheduler: weekday must be between 0 and 6")
		}

		return scheduler.Weekly{Day: time.Weekday(value.Day), Hour: value.Hour, Minute: value.Minute, TZ: location}, nil
	default:
		return nil, fmt.Errorf("scheduler: unsupported schedule type")
	}
}
