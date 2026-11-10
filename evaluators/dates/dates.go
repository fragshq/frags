/*
 * Copyright (C) 2026 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

// Package dates provides the date manipulation primitives exposed to the template and expression engines.
package dates

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	// embeds the IANA timezone database, so that timezones resolve on every platform (notably Windows).
	_ "time/tzdata"
)

// Time wraps time.Time so that, when rendered without explicit formatting, it prints as RFC3339.
type Time struct{ time.Time }

// String returns the RFC3339 representation of the time.
func (t Time) String() string {
	return t.Time.Format(time.RFC3339)
}

// layoutPresets maps friendly names to Go layouts.
var layoutPresets = map[string]string{
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
	"DateTime":    time.DateTime,
	"DateOnly":    time.DateOnly,
	"TimeOnly":    time.TimeOnly,
}

// parseLayouts are the layouts attempted, in order, when converting a string to a time.
var parseLayouts = []string{
	time.RFC3339Nano,
	time.DateTime,
	time.DateOnly,
}

// LoadLocation resolves a timezone name. An empty string resolves to UTC.
func LoadLocation(tz string) (*time.Location, error) {
	if tz == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return loc, nil
}

// Now returns the current time in the given timezone. An empty tz means UTC.
// The monotonic clock reading is stripped.
func Now(tz string) (time.Time, error) {
	loc, err := LoadLocation(tz)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Round(0).In(loc), nil
}

// ToTime converts a supported value to a time.Time. Supported values are time.Time, Time, *time.Time,
// strings (RFC3339, "2006-01-02 15:04:05", "2006-01-02") and numbers (Unix timestamps in seconds).
func ToTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case Time:
		return t.Time, nil
	case *time.Time:
		if t == nil {
			return time.Time{}, fmt.Errorf("cannot convert nil to time")
		}
		return *t, nil
	case *Time:
		if t == nil {
			return time.Time{}, fmt.Errorf("cannot convert nil to time")
		}
		return t.Time, nil
	case string:
		for _, layout := range parseLayouts {
			if parsed, err := time.Parse(layout, t); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse %q as a time", t)
	case int:
		return time.Unix(int64(t), 0).UTC(), nil
	case int32:
		return time.Unix(int64(t), 0).UTC(), nil
	case int64:
		return time.Unix(t, 0).UTC(), nil
	case uint:
		return time.Unix(int64(t), 0).UTC(), nil
	case uint32:
		return time.Unix(int64(t), 0).UTC(), nil
	case uint64:
		return time.Unix(int64(t), 0).UTC(), nil
	case float32:
		return floatToTime(float64(t)), nil
	case float64:
		return floatToTime(t), nil
	case nil:
		return time.Time{}, fmt.Errorf("cannot convert nil to time")
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time", v)
	}
}

// floatToTime converts a fractional Unix timestamp in seconds to a time.Time.
func floatToTime(f float64) time.Time {
	sec, frac := math.Modf(f)
	return time.Unix(int64(sec), int64(math.Round(frac*1e9))).UTC()
}

// Format formats a time-like value using a Go layout or a preset name. The special layouts "unix" and "unixms"
// return the Unix timestamp in seconds and milliseconds. If tz is not empty, the time is converted to that
// timezone before formatting, otherwise the time's own timezone is retained.
func Format(t any, layout string, tz string) (string, error) {
	tm, err := ToTime(t)
	if err != nil {
		return "", err
	}
	if tz != "" {
		loc, err := LoadLocation(tz)
		if err != nil {
			return "", err
		}
		tm = tm.In(loc)
	}
	switch layout {
	case "unix":
		return strconv.FormatInt(tm.Unix(), 10), nil
	case "unixms":
		return strconv.FormatInt(tm.UnixMilli(), 10), nil
	}
	if preset, ok := layoutPresets[layout]; ok {
		layout = preset
	}
	if layout == "" {
		return "", fmt.Errorf("empty date layout")
	}
	return tm.Format(layout), nil
}

// Add adds a duration string (see ParseDuration) to a time-like value. Days and weeks are added on the calendar
// (same wall clock time), so they are DST-safe. A negative duration subtracts.
func Add(t any, duration string) (time.Time, error) {
	tm, err := ToTime(t)
	if err != nil {
		return time.Time{}, err
	}
	days, rest, err := parseDuration(duration)
	if err != nil {
		return time.Time{}, err
	}
	return tm.AddDate(0, 0, days).Add(rest), nil
}

// ParseDuration parses a duration string. It supports all the units of time.ParseDuration plus "d" (days) and
// "w" (weeks), which must be integers. Days are considered to be 24h long.
// Examples: "1h30m", "-7d", "1w2d3h".
func ParseDuration(s string) (time.Duration, error) {
	days, rest, err := parseDuration(s)
	if err != nil {
		return 0, err
	}
	return time.Duration(days)*24*time.Hour + rest, nil
}

// parseDuration splits a duration string into a number of calendar days and a remainder duration.
func parseDuration(s string) (int, time.Duration, error) {
	orig := s
	if s == "" {
		return 0, 0, fmt.Errorf("invalid duration %q", orig)
	}
	sign := 1
	if s[0] == '-' || s[0] == '+' {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	if s == "0" {
		return 0, 0, nil
	}
	if s == "" {
		return 0, 0, fmt.Errorf("invalid duration %q", orig)
	}
	days := 0
	var rest strings.Builder
	for s != "" {
		i := 0
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
			i++
		}
		if i == 0 {
			return 0, 0, fmt.Errorf("invalid duration %q", orig)
		}
		num := s[:i]
		s = s[i:]
		j := 0
		for j < len(s) && !(s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
			j++
		}
		if j == 0 {
			return 0, 0, fmt.Errorf("missing unit in duration %q", orig)
		}
		unit := s[:j]
		s = s[j:]
		switch unit {
		case "d", "w":
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, 0, fmt.Errorf("days and weeks must be integers in duration %q", orig)
			}
			if unit == "w" {
				n *= 7
			}
			days += n
		default:
			rest.WriteString(num)
			rest.WriteString(unit)
		}
	}
	var d time.Duration
	if rest.Len() > 0 {
		var err error
		d, err = time.ParseDuration(rest.String())
		if err != nil {
			return 0, 0, fmt.Errorf("invalid duration %q: %w", orig, err)
		}
	}
	return sign * days, time.Duration(sign) * d, nil
}
