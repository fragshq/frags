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
	"bytes"
	"errors"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/expr-lang/expr"
	"github.com/fragshq/frags/evaluators/dates"
	"github.com/fragshq/frags/util"
	"github.com/fragshq/zealql"
)

const (
	ParamsAttr     = "params"
	ContextAttr    = "context"
	ComponentsAttr = "components"
	IteratorAttr   = "it"
	VarsAttr       = "vars"
	DbAttr         = "db"
)

// EvalScope is the scope for evaluating expressions.
type EvalScope map[string]any

// WithIterator clones the scope and adds the iterator.
func (e EvalScope) WithIterator(it any) EvalScope {
	e[IteratorAttr] = it
	return e
}

// WithVars clones the scope and adds the vars.
func (e EvalScope) WithVars(vars map[string]any) EvalScope {
	if vars == nil {
		return e
	}

	for k, v := range vars {
		e[VarsAttr].(map[string]any)[k] = v
	}
	return e
}

func (e EvalScope) WithDB(db *zealql.Database) EvalScope {
	e[DbAttr] = db
	return e
}

func (e EvalScope) WithParams(params map[string]any) EvalScope {
	if params == nil {
		e[ParamsAttr] = make(map[string]any)
	}
	for k, v := range params {
		e[ParamsAttr].(map[string]any)[k] = v
	}
	return e
}

// Vars returns the vars map.
func (e EvalScope) Vars() map[string]any {
	return e[VarsAttr].(map[string]any)
}

// NewEvalScope is the EvalScope constructor, unbounded to a specific Runner.
func NewEvalScope() EvalScope {
	return EvalScope{
		ParamsAttr:     make(map[string]any),
		ContextAttr:    make(map[string]any),
		ComponentsAttr: make(map[string]any),
		VarsAttr:       make(map[string]any),
	}
}

// EvaluateTemplate evaluates a Golang template with the given scope.
func EvaluateTemplate(text string, scope EvalScope) (string, error) {
	for i := 0; i < 3; i++ {
		if scope == nil || !strings.Contains(text, "{{") {
			return text, nil
		}
		tmpl := template.New("tpl").Funcs(templateFuncs)
		parsedTmpl, err := tmpl.Parse(text)
		if err != nil {
			return text, err
		}
		writer := bytes.NewBufferString("")
		err = parsedTmpl.Execute(writer, map[string]any(scope))
		if err != nil {
			return text, err
		}
		text = writer.String()
	}
	return text, nil
}

func EvaluateExpression(expression string, scope EvalScope) (any, error) {
	c, err := expr.Compile(expression, append(exprFunctions(), expr.Env(scope))...)
	if err != nil {
		return nil, err
	}
	return expr.Run(c, map[string]any(scope))
}

// EvaluateBooleanExpression evaluates a boolean expression with the given scope using expr.
func EvaluateBooleanExpression(expression string, scope EvalScope) (bool, error) {
	c, err := expr.Compile(expression, append(exprFunctions(), expr.Env(scope))...)
	if err != nil {
		return false, err
	}
	res, err := expr.Run(c, map[string]any(scope))
	if err != nil {
		return false, err
	}
	if b, ok := res.(bool); ok {
		return b, nil
	}
	return false, errors.New("return type is not a boolean")
}

// EvaluateArrayExpression evaluates an array expression, expecting the target to be an array.
func EvaluateArrayExpression(expression string, scope EvalScope) ([]any, error) {
	c, err := expr.Compile(expression, append(exprFunctions(), expr.Env(scope))...)
	if err != nil {
		return nil, err
	}
	res, err := expr.Run(c, map[string]any(scope))
	if err != nil {
		return nil, err
	}
	rv := util.ToConcreteValue(reflect.ValueOf(res))
	if rv.Kind() == reflect.Slice {
		result := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, nil
	} else {
		return nil, errors.New("expression did not evaluate to an array")
	}
}

// EvaluateMapValues will evaluate all **first level** strings as templates in a given map of arguments.
func EvaluateMapValues(args map[string]any, scope EvalScope) (map[string]any, error) {
	if args == nil {
		return make(map[string]any), nil
	}
	out := make(map[string]any)
	for k, v := range args {
		if s, ok := v.(string); ok {
			if ref, ok := ExtractVarRef(s); ok {
				res, err := EvaluateExpression(ref, scope)
				if err != nil {
					return nil, err
				}
				out[k] = res
			} else {
				res, err := EvaluateTemplate(s, scope)
				if err != nil {
					return nil, err
				}
				out[k] = res
			}
		} else {
			out[k] = v
		}
	}
	return out, nil
}

// templateFuncs are the functions available in the templates.
var templateFuncs = template.FuncMap{
	"json": func(v any) string {
		return util.MustJsonIndentString(v)
	},
	"now": func(tz ...string) (dates.Time, error) {
		zone, err := optionalTz(tz)
		if err != nil {
			return dates.Time{}, err
		}
		t, err := dates.Now(zone)
		return dates.Time{Time: t}, err
	},
	"dateFormat": func(t any, layout string, tz ...string) (string, error) {
		zone, err := optionalTz(tz)
		if err != nil {
			return "", err
		}
		return dates.Format(t, layout, zone)
	},
	"dateAdd": func(t any, duration string) (dates.Time, error) {
		res, err := dates.Add(t, duration)
		return dates.Time{Time: res}, err
	},
}

// optionalTz extracts the optional timezone from variadic template function arguments.
func optionalTz(tz []string) (string, error) {
	switch len(tz) {
	case 0:
		return "", nil
	case 1:
		return tz[0], nil
	default:
		return "", errors.New("at most one timezone argument is allowed")
	}
}

type Vars map[string]any

func (v *Vars) Apply(data map[string]any) {
	for k, val := range data {
		(*v)[k] = val
	}
}

func exprFunctions() []expr.Option {
	return []expr.Option{
		expr.Function("unique",
			func(params ...any) (any, error) {
				seen := make(map[any]bool)
				result := make([]any, 0)
				input, ok := params[0].([]any)
				if !ok {
					return nil, errors.New("unique function expects an array as input")
				}
				for _, item := range input {
					if !seen[item] {
						seen[item] = true
						result = append(result, item)
					}
				}
				return result, nil
			}, new(func([]any) []any)),
		expr.Function("chunk",
			func(params ...any) (any, error) {
				chunks := make([]any, 0)
				input, ok := params[0].([]any)
				if !ok {
					return nil, errors.New("chunk function expects an array as input")
				}
				size, ok := params[1].(int)
				if !ok {
					return nil, errors.New("chunk function expects an integer as second parameter")
				}
				for i := 0; i < len(input); i += size {
					end := i + size
					if end > len(input) {
						end = len(input)
					}
					chunks = append(chunks, input[i:end])
				}
				return chunks, nil
			}, new(func([]any, int) []any)),
		expr.Function("render",
			func(params ...any) (any, error) {
				return EvaluateTemplate(params[0].(string), params[1].(map[string]any))
			}, new(func(string, EvalScope) (string, error))),
		expr.Function("now",
			func(params ...any) (any, error) {
				tz := ""
				if len(params) > 0 {
					tz = params[0].(string)
				}
				return dates.Now(tz)
			},
			new(func() time.Time),
			new(func(string) (time.Time, error))),
		expr.Function("dateFormat",
			func(params ...any) (any, error) {
				tz := ""
				if len(params) > 2 {
					tz = params[2].(string)
				}
				return dates.Format(params[0], params[1].(string), tz)
			},
			new(func(any, string) (string, error)),
			new(func(any, string, string) (string, error))),
		expr.Function("dateAdd",
			func(params ...any) (any, error) {
				return dates.Add(params[0], params[1].(string))
			},
			new(func(any, string) (time.Time, error))),
	}
}

func ExtractVarRef(s string) (string, bool) {
	if !strings.HasPrefix(s, "$(") || !strings.HasSuffix(s, ")") {
		return "", false
	}

	if len(s) < 3 {
		return "", false
	}

	depth := 1
	i := 2

	for i < len(s)-1 {
		if s[i] == '\\' && i+1 < len(s) {
			i += 2
			continue
		}

		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return "", false
			}
		}
		i++
	}
	if depth == 1 && s[len(s)-1] == ')' {
		return s[2 : len(s)-1], true
	}

	return "", false
}
