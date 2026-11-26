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
	"strings"
	"testing"
	"time"

	"github.com/fragshq/frags/resources"
	"github.com/fragshq/frags/util"
	"github.com/stretchr/testify/assert"
)

func exposedNames(r ResolvedTools) []string {
	names := make([]string, 0)
	for _, rt := range r {
		names = append(names, rt.ExposedName)
	}
	return names
}

func TestToolDefinitions_Resolve_AlwaysPrefixed(t *testing.T) {
	// a single server: no clash, but names are prefixed anyway
	res, err := ToolDefinitions{{Name: "hr", Type: ToolTypeMCP}}.Resolve(clashingFunctions())
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"hr_list_users", "hr_get_user"}, exposedNames(res))
}

func TestToolDefinitions_Resolve_Clash(t *testing.T) {
	fx := clashingFunctions()
	fx = fx.WithFunctions(ExternalFunctions{"list_users": {Name: "list_users", Collection: "crm", Description: "crm users"}})
	defs := ToolDefinitions{{Name: "crm", Type: ToolTypeMCP}, {Name: "hr", Type: ToolTypeMCP}}
	res, err := defs.Resolve(fx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"crm_list_users", "hr_list_users", "hr_get_user"}, exposedNames(res))

	rt, ok := res.Lookup("crm_list_users")
	assert.True(t, ok)
	assert.Equal(t, "crm:list_users", rt.Function.QualifiedName())
	assert.Equal(t, "crm users", rt.Description)
}

func TestToolDefinitions_Resolve_NoDoublePrefix(t *testing.T) {
	fx := ExternalFunctions{}.WithFunctions(ExternalFunctions{
		"github_create_issue": {Name: "github_create_issue", Collection: "github"},
		"create_issue":        {Name: "create_issue", Collection: "github"},
	})
	res, err := ToolDefinitions{{Name: "github", Type: ToolTypeMCP}}.Resolve(fx)
	assert.NoError(t, err)
	names := exposedNames(res)
	assert.Len(t, names, 2)
	assert.NotEqual(t, names[0], names[1])
	assert.Contains(t, names, "github_create_issue")
	for _, n := range names {
		rt, ok := res.Lookup(n)
		assert.True(t, ok)
		assert.Equal(t, "github", rt.Function.Collection)
	}
}

func TestToolDefinitions_Resolve_Allowlist(t *testing.T) {
	for _, allowed := range []string{"list_users", "crm_list_users", "crm:list_users"} {
		defs := ToolDefinitions{
			{Name: "crm", Type: ToolTypeMCP, Allowlist: &[]string{allowed}},
			{Name: "hr", Type: ToolTypeMCP, Allowlist: &[]string{"hr:get_user"}},
		}
		res, err := defs.Resolve(clashingFunctions())
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"crm_list_users", "hr_get_user"}, exposedNames(res), allowed)
	}
}

func TestToolDefinitions_Resolve_FunctionDefinitions(t *testing.T) {
	fx := clashingFunctions()

	// ambiguous bare name is a configuration error
	_, err := ToolDefinitions{{Name: "list_users", Type: ToolTypeFunction}}.Resolve(fx)
	assert.ErrorContains(t, err, "ambiguous")

	// missing functions are skipped, as before
	res, err := ToolDefinitions{{Name: "nope", Type: ToolTypeFunction}}.Resolve(fx)
	assert.NoError(t, err)
	assert.Empty(t, res)

	// prefixed name resolves too
	res, err = ToolDefinitions{{Name: "crm_list_users", Type: ToolTypeFunction}}.Resolve(fx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"crm_list_users"}, exposedNames(res))

	// qualified name + override, deduped against the MCP definition exposing the same function
	res, err = ToolDefinitions{
		{Name: "hr", Type: ToolTypeMCP},
		{Name: "hr:list_users", Type: ToolTypeFunction, Description: "override"},
	}.Resolve(fx)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	rt, ok := res.Lookup("hr_list_users")
	assert.True(t, ok)
	assert.Equal(t, "override", rt.Description)
}

func TestToolDefinitions_Resolve_Apicp(t *testing.T) {
	fx := clashingFunctions().WithFunctions(ExternalFunctions{
		"list_deals": {Name: "list_deals", Collection: "deals"},
		"get_deal":   {Name: "get_deal", Collection: "deals"},
	})
	res, err := ToolDefinitions{{Name: "deals", Type: ToolTypeApicp}}.Resolve(fx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"deals_list_deals", "deals_get_deal"}, exposedNames(res))

	res, err = ToolDefinitions{{Name: "deals", Type: ToolTypeApicp, Allowlist: &[]string{"deals_get_deal"}}}.Resolve(fx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"deals_get_deal"}, exposedNames(res))
}

func TestResolvedTools_Run(t *testing.T) {
	runner := NewRunner(NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi(),
		WithExternalFunctions(clashingFunctions()))
	ctx := util.NewFragsContext(time.Minute)

	res, err := ToolDefinitions{{Name: "crm", Type: ToolTypeMCP}}.Resolve(clashingFunctions())
	assert.NoError(t, err)
	out, err := res.Run(ctx, "crm_list_users", nil, &runner)
	assert.NoError(t, err)
	assert.Equal(t, "crm", out)

	res, err = ToolDefinitions{{Name: "crm", Type: ToolTypeMCP}, {Name: "hr", Type: ToolTypeMCP}}.Resolve(clashingFunctions())
	assert.NoError(t, err)
	out, err = res.Run(ctx, "hr_list_users", nil, &runner)
	assert.NoError(t, err)
	assert.Equal(t, "hr", out)
}

// TestResolvedTools_Run_HashedName makes sure that names over the provider limit, which get truncated and hashed,
// still dispatch to the right function, even when two of them share the same truncated stem
func TestResolvedTools_Run_HashedName(t *testing.T) {
	server := "Barndoor-SalesForce-Something"
	fx := ExternalFunctions{}
	for _, name := range []string{"search_opportunities_by_account_owner", "search_opportunities_by_account_region"} {
		f := ExternalFunction{Name: name, Collection: server, Func: func(ctx *util.FragsContext, data map[string]any) (any, error) {
			return name, nil
		}}
		fx[f.QualifiedName()] = f
	}
	runner := NewRunner(NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi(), WithExternalFunctions(fx))
	ctx := util.NewFragsContext(time.Minute)

	res, err := ToolDefinitions{{Name: server, Type: ToolTypeMCP}}.Resolve(fx)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	again, err := ToolDefinitions{{Name: server, Type: ToolTypeMCP}}.Resolve(fx)
	assert.NoError(t, err)
	for i, rt := range res {
		assert.Len(t, rt.ExposedName, maxLlmNameLength)
		assert.Equal(t, again[i].ExposedName, rt.ExposedName, "hashed names must be stable across requests")
		out, err := res.Run(ctx, rt.ExposedName, nil, &runner)
		assert.NoError(t, err)
		assert.Equal(t, rt.Function.Name, out)
	}
	assert.NotEqual(t, res[0].ExposedName, res[1].ExposedName)
}

func TestLlmName(t *testing.T) {
	assert.Equal(t, "crm_list_users", llmName(ExternalFunction{Name: "list_users", Collection: "crm"}))
	assert.Equal(t, "github_create_issue", llmName(ExternalFunction{Name: "github_create_issue", Collection: "github"}))
	assert.Equal(t, "my_server_list_users", llmName(ExternalFunction{Name: "list_users", Collection: "my.server"}))
	assert.Equal(t, "list_users", llmName(ExternalFunction{Name: "list_users"}))
	long := llmName(ExternalFunction{Name: "list_users", Collection: strings.Repeat("x", 80)})
	assert.Len(t, long, maxLlmNameLength)
	assert.NotEqual(t, long, llmName(ExternalFunction{Name: "list_users", Collection: strings.Repeat("x", 81)}))
}
