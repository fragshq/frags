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
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/fragshq/frags/schema"
	"github.com/fragshq/frags/util"
)

// maxLlmNameLength is the maximum tool name length most providers accept
const maxLlmNameLength = 64

// ResolvedTool is a function, as it gets exposed to the LLM in a specific request.
// ExposedName is the name the LLM sees, and uses to call the function.
// Function is the actual function.
// Description, Schema and OutputSchema are what the LLM sees, after applying any override from the tool definition.
type ResolvedTool struct {
	ExposedName  string
	Function     ExternalFunction
	Description  string
	Schema       *schema.Schema
	OutputSchema *schema.Schema
}

// ResolvedTools is the set of functions exposed to the LLM in a specific request
type ResolvedTools []ResolvedTool

// Resolve expands the function, MCP, collection and apicp tool definitions into the concrete functions to expose to the
// LLM. Functions are always exposed with their collection prefix (see llmName), so the name the LLM sees does not
// depend on which other functions are in the request.
// Function tool definitions that can't be found are skipped, while ambiguous ones return an error.
func (t ToolDefinitions) Resolve(functions ExternalFunctions) (ResolvedTools, error) {
	res := make(ResolvedTools, 0)
	seen := make(map[string]bool)
	// function definitions go first, so that their description/schema overrides win over the same function being
	// exposed by an MCP or collection definition.
	for _, tool := range t {
		if tool.Type != ToolTypeFunction {
			continue
		}
		fx, err := functions.Resolve(tool.Name)
		if err != nil {
			var notFound *ErrFunctionNotFound
			if errors.As(err, &notFound) {
				continue
			}
			return nil, err
		}
		if seen[fx.QualifiedName()] {
			continue
		}
		seen[fx.QualifiedName()] = true
		rt := ResolvedTool{
			Function:     fx,
			Description:  fx.Description,
			Schema:       fx.Schema,
			OutputSchema: fx.OutputSchema,
		}
		if tool.InputSchema != nil {
			rt.Schema = tool.InputSchema
		}
		if len(tool.Description) > 0 {
			rt.Description = tool.Description
		}
		res = append(res, rt)
	}
	for _, tool := range t {
		if tool.Type != ToolTypeMCP && tool.Type != ToolTypeCollection && tool.Type != ToolTypeApicp {
			continue
		}
		collection := functions.ListByCollection(tool.Name)
		for _, k := range slices.Sorted(maps.Keys(collection)) {
			fx := collection[k]
			if seen[fx.QualifiedName()] {
				continue
			}
			if tool.Allowlist != nil && !slices.Contains(*tool.Allowlist, k) &&
				!slices.ContainsFunc(fx.Names(), func(n string) bool { return slices.Contains(*tool.Allowlist, n) }) {
				continue
			}
			seen[fx.QualifiedName()] = true
			res = append(res, ResolvedTool{
				Function:     fx,
				Description:  fx.Description,
				Schema:       fx.Schema,
				OutputSchema: fx.OutputSchema,
			})
		}
	}
	res.assignExposedNames()
	return res, nil
}

// assignExposedNames sets the name the LLM will see for each function (see llmName). If two functions end up with
// the same name, a hash is appended to keep them apart.
func (r ResolvedTools) assignExposedNames() {
	used := make(map[string]bool)
	for i := range r {
		fx := r[i].Function
		r[i].ExposedName = llmName(fx)
		if used[r[i].ExposedName] {
			r[i].ExposedName = withHash(r[i].ExposedName, fx.QualifiedName())
		}
		used[r[i].ExposedName] = true
	}
}

// Lookup finds a resolved tool by the name the LLM used to call it
func (r ResolvedTools) Lookup(exposedName string) (ResolvedTool, bool) {
	for _, rt := range r {
		if rt.ExposedName == exposedName {
			return rt, true
		}
	}
	return ResolvedTool{}, false
}

// Run runs the function the LLM called by its exposed name. If the name isn't among the exposed ones, it falls back
// to the runner global resolution.
func (r ResolvedTools) Run(ctx *util.FragsContext, exposedName string, args map[string]any, runner ExportableRunner) (any, error) {
	if rt, ok := r.Lookup(exposedName); ok {
		return rt.Function.Run(ctx, args, runner)
	}
	return runner.RunFunction(ctx, exposedName, args)
}

// llmName builds the provider-safe name the LLM sees for a function: always prefixed by the collection
// (collection_name), unless the function name already carries that prefix.
func llmName(f ExternalFunction) string {
	name := f.Name
	if f.Collection != "" && !strings.HasPrefix(f.Name, f.Collection+PrefixedNameSeparator) {
		name = f.PrefixedName()
	}
	name = sanitizeLlmName(name)
	if len(name) > maxLlmNameLength {
		return withHash(name, f.QualifiedName())
	}
	return name
}

// withHash truncates the name if needed and appends a short hash of the seed, to keep it unique
func withHash(name string, seed string) string {
	sum := sha1.Sum([]byte(seed))
	suffix := "_" + hex.EncodeToString(sum[:])[:8]
	if len(name)+len(suffix) > maxLlmNameLength {
		name = name[:maxLlmNameLength-len(suffix)]
	}
	return name + suffix
}

// sanitizeLlmName replaces any character providers don't accept in tool names with an underscore
func sanitizeLlmName(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}
