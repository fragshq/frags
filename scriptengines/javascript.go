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
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/fragshq/frags"
	"github.com/fragshq/frags/evaluators/dates"
	"github.com/fragshq/frags/util"
)

type JavascriptScriptingEngine struct {
}

func NewJavascriptScriptingEngine() *JavascriptScriptingEngine {
	return &JavascriptScriptingEngine{}
}

func (e *JavascriptScriptingEngine) RunCode(ctx *util.FragsContext, code string, params any, runner frags.ExportableRunner) (any, error) {
	innerCtx := util.WithFragsContext(ctx, 1*time.Minute)
	defer innerCtx.Cancel(nil)
	vm := goja.New()
	go func() {
		<-innerCtx.Done()
		vm.Interrupt("timeout")
	}()
	var args any
	switch t := params.(type) {
	case []byte:
		args = string(t)
	default:
		args = params
	}
	if err := vm.Set("args", args); err != nil {
		return nil, err
	}
	if err := vm.Set("runFunction", func(name string, args map[string]any) any {
		res, err := runner.RunFunction(innerCtx, name, args)
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		return res
	}); err != nil {
		return nil, err
	}
	if err := registerDateFunctions(vm); err != nil {
		return nil, err
	}
	res, err := vm.RunString(code)
	if err != nil {
		return nil, err
	}
	return res.Export(), nil
}

// registerDateFunctions exposes now, dateFormat and dateAdd (see the evaluators/dates package) to the VM.
// Times are returned as native JS Date objects.
func registerDateFunctions(vm *goja.Runtime) error {
	toDate := func(t time.Time) goja.Value {
		d, err := vm.New(vm.Get("Date"), vm.ToValue(t.UnixMilli()))
		if err != nil {
			panic(err)
		}
		return d
	}
	fail := func(err error) {
		panic(vm.NewTypeError(err.Error()))
	}
	if err := vm.Set("now", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 1 {
			fail(errors.New("now: at most one timezone argument is allowed"))
		}
		tz := optionalStringArg(vm, call, 0, "now", "timezone")
		t, err := dates.Now(tz)
		if err != nil {
			fail(err)
		}
		return toDate(t)
	}); err != nil {
		return err
	}
	if err := vm.Set("dateFormat", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 || len(call.Arguments) > 3 {
			fail(errors.New("dateFormat: expects (t, layout, tz?)"))
		}
		layout := optionalStringArg(vm, call, 1, "dateFormat", "layout")
		tz := optionalStringArg(vm, call, 2, "dateFormat", "timezone")
		res, err := dates.Format(exportTime(call.Argument(0)), layout, tz)
		if err != nil {
			fail(err)
		}
		return vm.ToValue(res)
	}); err != nil {
		return err
	}
	return vm.Set("dateAdd", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			fail(errors.New("dateAdd: expects (t, duration)"))
		}
		duration := optionalStringArg(vm, call, 1, "dateAdd", "duration")
		t, err := dates.Add(exportTime(call.Argument(0)), duration)
		if err != nil {
			fail(err)
		}
		return toDate(t)
	})
}

// exportTime exports a JS value for the dates package. JS Dates carry no timezone, and goja exports them in the
// machine's local zone: they are normalized to UTC so results don't depend on the host.
func exportTime(v goja.Value) any {
	exported := v.Export()
	if t, ok := exported.(time.Time); ok {
		return t.UTC()
	}
	return exported
}

// optionalStringArg returns the string argument at index i, or an empty string if it is missing or undefined.
// It panics with a TypeError if the argument is not a string.
func optionalStringArg(vm *goja.Runtime, call goja.FunctionCall, i int, fn string, name string) string {
	arg := call.Argument(i)
	if goja.IsUndefined(arg) || goja.IsNull(arg) {
		return ""
	}
	s, ok := arg.Export().(string)
	if !ok {
		panic(vm.NewTypeError(fmt.Sprintf("%s: %s must be a string", fn, name)))
	}
	return s
}
