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

package scriptengines

import (
	"testing"
	"time"

	"github.com/fragshq/frags/util"
	"github.com/stretchr/testify/assert"
)

func runJS(t *testing.T, code string) (any, error) {
	t.Helper()
	ctx := util.NewFragsContext(10 * time.Second)
	defer ctx.Cancel(nil)
	return NewJavascriptScriptingEngine().RunCode(ctx, code, map[string]any{"ts": "2026-03-28T10:30:00Z"}, nil)
}

func TestJavascript_DateFunctions(t *testing.T) {
	cases := []struct {
		code string
		want any
	}{
		{`dateFormat(args.ts, "DateOnly")`, "2026-03-28"},
		{`dateFormat(args.ts, "DateTime", "Europe/Rome")`, "2026-03-28 11:30:00"},
		{`dateFormat(dateAdd(args.ts, "1w2h"), "DateTime")`, "2026-04-04 12:30:00"},
		{`dateAdd(args.ts, "-1d").toISOString()`, "2026-03-27T10:30:00.000Z"},
		{`dateAdd(args.ts, "1h") - new Date(args.ts)`, int64(3600000)},
		{`dateFormat(new Date("2026-03-28T10:30:00Z"), "RFC3339")`, "2026-03-28T10:30:00Z"},
		{`dateAdd(args.ts, "1d") instanceof Date`, true},
		{`now() instanceof Date`, true},
		{`now("Europe/Rome") <= now()`, true},
		{`dateFormat(1774693800, "DateOnly")`, "2026-03-28"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			res, err := runJS(t, c.code)
			assert.NoError(t, err)
			assert.Equal(t, c.want, res)
		})
	}
	t.Run("now is current", func(t *testing.T) {
		res, err := runJS(t, `now().getTime()`)
		assert.NoError(t, err)
		assert.InDelta(t, time.Now().UnixMilli(), res, 2000)
	})
	t.Run("errors are catchable", func(t *testing.T) {
		res, err := runJS(t, `try { dateAdd(args.ts, "nope"); "no error" } catch (e) { e instanceof TypeError }`)
		assert.NoError(t, err)
		assert.Equal(t, true, res)
	})
	t.Run("errors", func(t *testing.T) {
		for _, code := range []string{
			`now("Mars/Olympus")`,
			`now("UTC", "UTC")`,
			`dateFormat(args.ts)`,
			`dateFormat(args.ts, 42)`,
			`dateFormat("garbage", "DateOnly")`,
			`dateAdd(args.ts)`,
			`dateAdd(undefined, "1d")`,
		} {
			_, err := runJS(t, code)
			assert.Error(t, err, code)
		}
	})
}
