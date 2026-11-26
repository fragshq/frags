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

package frags

import (
	"testing"
	"time"

	"github.com/fragshq/frags/resources"
	"github.com/fragshq/frags/util"
	"github.com/stretchr/testify/assert"
)

func TestExternalFunction_Run(t *testing.T) {
	runner := NewRunner(NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi())
	ef := ExternalFunction{
		Name:       "f1",
		Collection: "c1",
		Func: func(ctx *util.FragsContext, args map[string]any) (any, error) {
			return args, nil
		},
	}
	efc := ExternalFunctions{"f1": ef}
	runner.sessionManager.Transformers = &Transformers{
		{
			Name:            "t1",
			OnFunctionInput: util.Ptr("f1"),
			Expr:            util.Ptr("{\"yay\": args.foo}"),
		},
	}
	res, err := efc.ListByCollection("c1").Get("f1").Run(util.NewFragsContext(1*time.Minute), map[string]any{"foo": "bar"}, &runner)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"yay": "bar"}, res)
}

func returning(value string) func(ctx *util.FragsContext, args map[string]any) (any, error) {
	return func(ctx *util.FragsContext, args map[string]any) (any, error) {
		return value, nil
	}
}

func clashingFunctions() ExternalFunctions {
	return ExternalFunctions{}.
		WithFunctions(ExternalFunctions{"list_users": {Name: "list_users", Collection: "crm", Func: returning("crm")}}).
		WithFunctions(ExternalFunctions{
			"list_users": {Name: "list_users", Collection: "hr", Func: returning("hr")},
			"get_user":   {Name: "get_user", Collection: "hr", Func: returning("hr_get")},
		})
}

func TestExternalFunctions_WithFunctions_NoOverwrite(t *testing.T) {
	fx := clashingFunctions()
	assert.Len(t, fx, 3)
	assert.Contains(t, fx, "crm:list_users")
	assert.Contains(t, fx, "hr:list_users")
	assert.Contains(t, fx, "hr:get_user")
}

func TestExternalFunctions_Resolve(t *testing.T) {
	fx := clashingFunctions()

	f, err := fx.Resolve("crm:list_users")
	assert.NoError(t, err)
	assert.Equal(t, "crm", f.Collection)

	f, err = fx.Resolve("get_user")
	assert.NoError(t, err)
	assert.Equal(t, "hr:get_user", f.QualifiedName())

	_, err = fx.Resolve("list_users")
	var ambiguous *ErrAmbiguousFunction
	assert.ErrorAs(t, err, &ambiguous)
	assert.Equal(t, []string{"crm:list_users", "hr:list_users"}, ambiguous.Candidates)

	_, err = fx.Resolve("nope")
	var notFound *ErrFunctionNotFound
	assert.ErrorAs(t, err, &notFound)

	assert.Equal(t, map[string][]string{"list_users": {"crm:list_users", "hr:list_users"}}, fx.Collisions())
}

func TestExternalFunctions_Resolve_LegacyKeys(t *testing.T) {
	// maps built by hand, keyed by bare name, keep working
	fx := ExternalFunctions{"f1": {Collection: "c1", Func: returning("ok")}}
	f, err := fx.Resolve("f1")
	assert.NoError(t, err)
	assert.Equal(t, "f1", f.Name)
	f, err = fx.Resolve("c1:f1")
	assert.NoError(t, err)
	assert.Equal(t, "f1", f.Name)
}

func TestRunner_RunFunction_Qualified(t *testing.T) {
	runner := NewRunner(NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi(),
		WithExternalFunctions(clashingFunctions()))
	ctx := util.NewFragsContext(time.Minute)

	res, err := runner.RunFunction(ctx, "hr:list_users", nil)
	assert.NoError(t, err)
	assert.Equal(t, "hr", res)

	res, err = runner.RunFunction(ctx, "get_user", nil)
	assert.NoError(t, err)
	assert.Equal(t, "hr_get", res)

	_, err = runner.RunFunction(ctx, "list_users", nil)
	assert.ErrorContains(t, err, "ambiguous")
}

func TestExternalFunction_Run_TransformerTriggers(t *testing.T) {
	runner := NewRunner(NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi())
	runner.sessionManager.Transformers = &Transformers{
		{Name: "bare", OnFunctionOutput: util.Ptr("list_users"), Expr: util.Ptr("args + \"+bare\"")},
		{Name: "qualified", OnFunctionOutput: util.Ptr("hr:list_users"), Expr: util.Ptr("args + \"+hr\"")},
	}
	fx := clashingFunctions()
	ctx := util.NewFragsContext(time.Minute)

	res, err := fx.Get("crm:list_users").Run(ctx, map[string]any{}, &runner)
	assert.NoError(t, err)
	assert.Equal(t, "crm+bare", res)

	res, err = fx.Get("hr:list_users").Run(ctx, map[string]any{}, &runner)
	assert.NoError(t, err)
	assert.Equal(t, "hr+bare+hr", res)
}

func TestExternalFunctions_Resolve_Prefixed(t *testing.T) {
	fx := clashingFunctions()

	f, err := fx.Resolve("crm_list_users")
	assert.NoError(t, err)
	assert.Equal(t, "crm:list_users", f.QualifiedName())

	f, err = fx.Resolve("hr_get_user")
	assert.NoError(t, err)
	assert.Equal(t, "hr:get_user", f.QualifiedName())

	// a function literally named like another one's prefixed name: ambiguous, no form wins
	fx = fx.WithFunctions(ExternalFunctions{"crm_list_users": {Name: "crm_list_users", Collection: "other"}})
	_, err = fx.Resolve("crm_list_users")
	var ambiguous *ErrAmbiguousFunction
	assert.ErrorAs(t, err, &ambiguous)
	assert.Equal(t, []string{"crm:list_users", "other:crm_list_users"}, ambiguous.Candidates)
	assert.Equal(t, []string{"crm:list_users", "other:crm_list_users"}, fx.Collisions()["crm_list_users"])

	// qualified names are never ambiguous
	f, err = fx.Resolve("other:crm_list_users")
	assert.NoError(t, err)
	assert.Equal(t, "other", f.Collection)
}

func TestExternalFunction_Names(t *testing.T) {
	assert.Equal(t, []string{"list_users", "crm_list_users", "crm:list_users"},
		ExternalFunction{Name: "list_users", Collection: "crm"}.Names())
	assert.Equal(t, []string{"list_users"}, ExternalFunction{Name: "list_users"}.Names())
}

func TestExternalFunction_Run_PrefixedTransformerTrigger(t *testing.T) {
	runner := NewRunner(NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi())
	runner.sessionManager.Transformers = &Transformers{
		{Name: "prefixed", OnFunctionOutput: util.Ptr("hr_list_users"), Expr: util.Ptr("args + \"+prefixed\"")},
	}
	fx := clashingFunctions()
	ctx := util.NewFragsContext(time.Minute)

	res, err := fx.Get("hr:list_users").Run(ctx, map[string]any{}, &runner)
	assert.NoError(t, err)
	assert.Equal(t, "hr+prefixed", res)

	res, err = fx.Get("crm:list_users").Run(ctx, map[string]any{}, &runner)
	assert.NoError(t, err)
	assert.Equal(t, "crm", res)
}
