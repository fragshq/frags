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

package evaluators

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type s2 struct {
	P1 float64 `json:"p1"`
	P2 float64 `json:"p2"`
}

type s3 struct {
	M1 float64 `json:"m1"`
	M2 float64 `json:"m2"`
}

type s1 struct {
	S2        s2            `json:"s2"`
	ArrayOfS2 []s2          `json:"arrayOfS2"`
	MapOfS3   map[string]s3 `json:"mapOfS3"`
}

func TestEvaluateArrayExpression(t *testing.T) {
	t.Run("successfully evaluates an array expression", func(t *testing.T) {
		res, err := EvaluateArrayExpression(`bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.NoError(t, err)
		assert.Equal(t, "foo", res[0])

	})
	t.Run("returns an error if the data does not resolve to an array", func(t *testing.T) {
		_, err := EvaluateArrayExpression(`bar`, EvalScope{
			"bar": 123,
		})
		assert.Error(t, err)
	})

	t.Run("returns correct data type for the selected array", func(t *testing.T) {
		res, err := EvaluateArrayExpression(`val`, EvalScope{
			"val": []s1{
				{
					S2: s2{
						P1: 123,
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, float64(123), res[0].(s1).S2.P1)
	})
}

func TestEvaluateBooleanExpression(t *testing.T) {
	t.Run("evaluates a true boolean expression", func(t *testing.T) {
		res, err := EvaluateBooleanExpression(`"foo" in bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.NoError(t, err)
		assert.True(t, res)
	})

	t.Run("evaluates a false boolean expression", func(t *testing.T) {
		res, err := EvaluateBooleanExpression(`"fuz" in bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.NoError(t, err)
		assert.False(t, res)
	})

	t.Run("fails to evaluate boolean", func(t *testing.T) {
		res, err := EvaluateBooleanExpression(`bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.Error(t, err)
		assert.False(t, res)
	})

}

func TestEvaluateTemplate(t *testing.T) {
	res, err := EvaluateTemplate(`{{.foo}}`, EvalScope{
		"foo": "bar",
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", res)

	res, err = EvaluateTemplate(`{{.foo}}`, EvalScope{
		"foo":  "{{.buzz}}",
		"buzz": "bar",
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", res)
}

func TestEvalScope_WithVars(t *testing.T) {
	scope := NewEvalScope()
	scope = scope.WithVars(map[string]any{
		"buzz": "baz",
	})
	assert.Equal(t, "baz", scope.Vars()["buzz"])

	scope = scope.WithVars(map[string]any{
		"hey": "joe",
	})
	assert.Equal(t, "baz", scope.Vars()["buzz"])
	assert.Equal(t, "joe", scope.Vars()["hey"])

	scope = scope.WithVars(map[string]any{
		"hey": "mario",
		"red": "ball",
	})
	assert.Equal(t, "baz", scope.Vars()["buzz"])
	assert.Equal(t, "mario", scope.Vars()["hey"])
	assert.Equal(t, "ball", scope.Vars()["red"])
}

func TestEvaluateExprFunctions(t *testing.T) {
	data := []any{map[string]any{"foo": "bar"}, map[string]any{"foo": "bar"}, map[string]any{"foo": "baz"}}
	arr, err := EvaluateArrayExpression("unique(map(data, .foo))", EvalScope{"data": data})
	assert.NoError(t, err)
	fmt.Println(arr)
}

func TestEvaluateExpression_Render(t *testing.T) {
	scope := NewEvalScope()
	scope.WithVars(map[string]any{
		"template": "What do we say? {{ .banana }}",
	})
	scope["context"] = map[string]any{"banana": "come monkey!"}
	out, err := EvaluateExpression("render(vars.template, context)", scope)
	assert.NoError(t, err)
	assert.Equal(t, "What do we say? come monkey!", out)
}

func TestTemplate_DateFunctions(t *testing.T) {
	scope := NewEvalScope()
	scope["context"] = map[string]any{"ts": "2026-03-28T10:30:00Z"}
	cases := []struct {
		tpl  string
		want string
	}{
		{`{{ dateFormat .context.ts "DateOnly" }}`, "2026-03-28"},
		{`{{ dateFormat .context.ts "DateTime" "Europe/Rome" }}`, "2026-03-28 11:30:00"},
		{`{{ dateAdd .context.ts "-1d" }}`, "2026-03-27T10:30:00Z"},
		{`{{ dateFormat (dateAdd .context.ts "1w2h") "DateTime" }}`, "2026-04-04 12:30:00"},
	}
	for _, c := range cases {
		t.Run(c.tpl, func(t *testing.T) {
			out, err := EvaluateTemplate(c.tpl, scope)
			assert.NoError(t, err)
			assert.Equal(t, c.want, out)
		})
	}
	t.Run("now", func(t *testing.T) {
		out, err := EvaluateTemplate(`{{ now }}`, scope)
		assert.NoError(t, err)
		parsed, err := time.Parse(time.RFC3339, out)
		assert.NoError(t, err)
		assert.WithinDuration(t, time.Now(), parsed, 2*time.Second)

		out, err = EvaluateTemplate(`{{ dateFormat (dateAdd now "1d") "DateOnly" "UTC" }}`, scope)
		assert.NoError(t, err)
		assert.Equal(t, time.Now().UTC().AddDate(0, 0, 1).Format(time.DateOnly), out)

		out, err = EvaluateTemplate(`{{ (now "Europe/Rome").Location }}`, scope)
		assert.NoError(t, err)
		assert.Equal(t, "Europe/Rome", out)
	})
	t.Run("errors", func(t *testing.T) {
		for _, tpl := range []string{
			`{{ now "Mars/Olympus" }}`,
			`{{ now "UTC" "UTC" }}`,
			`{{ dateAdd .context.ts "nope" }}`,
			`{{ dateFormat "garbage" "DateOnly" }}`,
		} {
			_, err := EvaluateTemplate(tpl, scope)
			assert.Error(t, err, tpl)
		}
	})
}

func TestExpression_DateFunctions(t *testing.T) {
	scope := NewEvalScope()
	scope["context"] = map[string]any{"ts": "2026-03-28T10:30:00Z"}
	cases := []struct {
		expr string
		want any
	}{
		{`dateFormat(context.ts, "DateOnly")`, "2026-03-28"},
		{`dateFormat(context.ts, "DateTime", "Europe/Rome")`, "2026-03-28 11:30:00"},
		{`dateFormat(dateAdd(context.ts, "1w2h"), "DateTime")`, "2026-04-04 12:30:00"},
		{`dateAdd(context.ts, "-1d")`, time.Date(2026, 3, 27, 10, 30, 0, 0, time.UTC)},
		{`dateAdd(context.ts, "1d") > date("2026-03-28T12:00:00Z")`, true},
		{`dateAdd(context.ts, "1h") - date("2026-03-28T10:30:00Z")`, time.Hour},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			out, err := EvaluateExpression(c.expr, scope)
			assert.NoError(t, err)
			assert.Equal(t, c.want, out)
		})
	}
	t.Run("now", func(t *testing.T) {
		out, err := EvaluateExpression(`now()`, scope)
		assert.NoError(t, err)
		assert.IsType(t, time.Time{}, out)
		assert.WithinDuration(t, time.Now(), out.(time.Time), 2*time.Second)

		out, err = EvaluateExpression(`now("Europe/Rome").Location().String()`, scope)
		assert.NoError(t, err)
		assert.Equal(t, "Europe/Rome", out)

		out, err = EvaluateBooleanExpression(`now() > dateAdd(now(), "-1h")`, scope)
		assert.NoError(t, err)
		assert.Equal(t, true, out)
	})
	t.Run("errors", func(t *testing.T) {
		for _, e := range []string{
			`now("Mars/Olympus")`,
			`dateAdd(context.ts, "nope")`,
			`dateFormat("garbage", "DateOnly")`,
			`dateFormat(context.ts, "DateOnly", "UTC", "extra")`,
		} {
			_, err := EvaluateExpression(e, scope)
			assert.Error(t, err, e)
		}
	})
}
