/*
 * Copyright (C) 2025 Simone Pezzano
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

package dates

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ref = time.Date(2026, 3, 28, 10, 30, 0, 0, time.UTC)

func TestToTime(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want time.Time
	}{
		{"time.Time", ref, ref},
		{"Time", Time{ref}, ref},
		{"*time.Time", &ref, ref},
		{"RFC3339 string", "2026-03-28T10:30:00Z", ref},
		{"DateTime string", "2026-03-28 10:30:00", ref},
		{"DateOnly string", "2026-03-28", time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)},
		{"int", int(ref.Unix()), ref},
		{"int64", ref.Unix(), ref},
		{"float64", float64(ref.Unix()) + 0.5, ref.Add(500 * time.Millisecond)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ToTime(c.in)
			assert.NoError(t, err)
			assert.True(t, c.want.Equal(got), "want %s, got %s", c.want, got)
		})
	}
	t.Run("invalid inputs", func(t *testing.T) {
		for _, in := range []any{"not a date", nil, true, []any{}} {
			_, err := ToTime(in)
			assert.Error(t, err, "%v", in)
		}
	})
}

func TestNow(t *testing.T) {
	t.Run("defaults to UTC", func(t *testing.T) {
		n, err := Now("")
		assert.NoError(t, err)
		assert.Equal(t, time.UTC, n.Location())
		assert.WithinDuration(t, time.Now(), n, time.Second)
	})
	t.Run("uses the given timezone", func(t *testing.T) {
		n, err := Now("Europe/Rome")
		assert.NoError(t, err)
		assert.Equal(t, "Europe/Rome", n.Location().String())
	})
	t.Run("strips the monotonic clock", func(t *testing.T) {
		n, _ := Now("")
		assert.NotContains(t, n.String(), "m=")
	})
	t.Run("invalid timezone", func(t *testing.T) {
		_, err := Now("Mars/Olympus")
		assert.Error(t, err)
	})
}

func TestFormat(t *testing.T) {
	cases := []struct {
		layout string
		tz     string
		want   string
	}{
		{"2006-01-02", "", "2026-03-28"},
		{"DateOnly", "", "2026-03-28"},
		{"DateTime", "", "2026-03-28 10:30:00"},
		{"RFC3339", "", "2026-03-28T10:30:00Z"},
		{"Kitchen", "", "10:30AM"},
		{"unix", "", "1774693800"},
		{"unixms", "", "1774693800000"},
		{"DateTime", "Europe/Rome", "2026-03-28 11:30:00"},
		{"RFC3339", "America/New_York", "2026-03-28T06:30:00-04:00"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s", c.layout, c.tz), func(t *testing.T) {
			got, err := Format(ref, c.layout, c.tz)
			assert.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
	t.Run("keeps the original timezone when tz is empty", func(t *testing.T) {
		rome, _ := time.LoadLocation("Europe/Rome")
		got, err := Format(ref.In(rome), "RFC3339", "")
		assert.NoError(t, err)
		assert.Equal(t, "2026-03-28T11:30:00+01:00", got)
	})
	t.Run("errors", func(t *testing.T) {
		_, err := Format(ref, "DateOnly", "Mars/Olympus")
		assert.Error(t, err)
		_, err = Format(ref, "", "")
		assert.Error(t, err)
		_, err = Format("garbage", "DateOnly", "")
		assert.Error(t, err)
	})
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"0", 0},
		{"90s", 90 * time.Second},
		{"1h30m", 90 * time.Minute},
		{"1d", 24 * time.Hour},
		{"-7d", -7 * 24 * time.Hour},
		{"+2h", 2 * time.Hour},
		{"1w2d3h", (9*24 + 3) * time.Hour},
		{"-1d12h", -36 * time.Hour},
		{"1.5h", 90 * time.Minute},
		{"500ms", 500 * time.Millisecond},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseDuration(c.in)
			assert.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
	t.Run("invalid", func(t *testing.T) {
		for _, in := range []string{"", "-", "abc", "10", "1.5d", "1x", "h1"} {
			_, err := ParseDuration(in)
			assert.Error(t, err, in)
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("adds and subtracts", func(t *testing.T) {
		got, err := Add(ref, "1d2h")
		assert.NoError(t, err)
		assert.Equal(t, ref.Add(26*time.Hour), got)
		got, err = Add(ref, "-1w")
		assert.NoError(t, err)
		assert.Equal(t, ref.AddDate(0, 0, -7), got)
	})
	t.Run("accepts strings", func(t *testing.T) {
		got, err := Add("2026-03-28T10:30:00Z", "30m")
		assert.NoError(t, err)
		assert.Equal(t, ref.Add(30*time.Minute), got)
	})
	t.Run("days are DST-safe", func(t *testing.T) {
		rome, _ := time.LoadLocation("Europe/Rome")
		// DST starts in Rome on 2026-03-29: the day is 23h long
		start := time.Date(2026, 3, 28, 10, 0, 0, 0, rome)
		got, err := Add(start, "1d")
		assert.NoError(t, err)
		assert.Equal(t, 10, got.Hour())
		assert.Equal(t, 29, got.Day())
	})
	t.Run("errors", func(t *testing.T) {
		_, err := Add(ref, "nope")
		assert.Error(t, err)
		_, err = Add("nope", "1d")
		assert.Error(t, err)
	})
}

func TestTime(t *testing.T) {
	tm := Time{ref}
	assert.Equal(t, "2026-03-28T10:30:00Z", tm.String())
	assert.Equal(t, "2026-03-28T10:30:00Z", fmt.Sprint(tm))
	data, err := json.Marshal(tm)
	assert.NoError(t, err)
	assert.Equal(t, `"2026-03-28T10:30:00Z"`, string(data))
}
