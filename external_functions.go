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
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fragshq/frags/log"
	"github.com/fragshq/frags/schema"
	"github.com/fragshq/frags/util"
)

// ExternalFunction represents a function that can be called by the AI model.
// Name is the function name.
// ToolsCollection is the MCP server or collection that contains the function.
// Description is the function description
// Schema is the input schema for the function.
type ExternalFunction struct {
	Func         func(ctx *util.FragsContext, data map[string]any) (any, error) `yaml:"-"`
	Name         string                                                         `yaml:"name"`
	Collection   string                                                         `yaml:"collection"`
	Description  string                                                         `yaml:"description"`
	Schema       *schema.Schema                                                 `yaml:"schema"`
	OutputSchema *schema.Schema                                                 `yaml:"outputSchema"`
}

func (f ExternalFunction) String() string {
	return fmt.Sprintf("%s:%s", f.Collection, f.Name)
}

// Run runs the function, applying any transformers defined in the runner.
func (f ExternalFunction) Run(ctx *util.FragsContext, args map[string]any, runner ExportableRunner) (any, error) {
	ax, err := runner.Transformers().FilterOnFunctionInput(f.Names()...).Transform(ctx, maps.Clone(args), runner)
	if err != nil {
		return nil, err
	}
	if !util.IsMapAny(ax) {
		return nil, fmt.Errorf("expected map[string]any, got %T", ax)
	}
	runner.Logger().Debug(log.NewEvent(log.StartEventType, log.FunctionComponent).WithFunction(fmt.Sprintf("%s(%v)", f.Name, ax)))
	out, err := f.Func(ctx, ax.(map[string]any))
	if err != nil {
		return out, err
	}

	out, err = runner.Transformers().FilterOnFunctionOutput(f.Names()...).Transform(ctx, out, runner)
	if err != nil {
		return nil, err
	}
	runner.Logger().Debug(log.NewEvent(log.EndEventType, log.FunctionComponent).
		WithFunction(fmt.Sprintf("%s(%v)", f.Name, util.MustJsonString(ax))).WithContent(out))
	return out, nil
}

// ExternalFunctions is a map of functions. When built via WithFunctions, keys are qualified names
// (see ExternalFunction.QualifiedName). Lookups should go through Resolve, which also understands bare names.
type ExternalFunctions map[string]ExternalFunction

// QualifiedNameSeparator separates the collection name from the function name in a qualified function name.
const QualifiedNameSeparator = ":"

// QualifiedName returns the fully qualified name of the function, in the form collection:name. If the function does
// not belong to any collection, the bare name is returned.
func (f ExternalFunction) QualifiedName() string {
	if f.Collection == "" {
		return f.Name
	}
	return f.Collection + QualifiedNameSeparator + f.Name
}

// PrefixedNameSeparator separates the collection name from the function name in a prefixed function name.
const PrefixedNameSeparator = "_"

// PrefixedName returns the function name prefixed by its collection, in the form collection_name. If the function
// does not belong to any collection, the bare name is returned.
func (f ExternalFunction) PrefixedName() string {
	if f.Collection == "" {
		return f.Name
	}
	return f.Collection + PrefixedNameSeparator + f.Name
}

// Names returns all the names the function can be referred by: bare, prefixed (collection_name) and qualified
// (collection:name). Duplicates are removed.
func (f ExternalFunction) Names() []string {
	return slices.Compact([]string{f.Name, f.PrefixedName(), f.QualifiedName()})
}

// matchesShortName returns true if the given name is the bare or the prefixed name of the function
func (f ExternalFunction) matchesShortName(name string) bool {
	return f.Name == name || f.PrefixedName() == name
}

// ErrFunctionNotFound is returned when no function matches the requested name
type ErrFunctionNotFound struct {
	Name string
}

func (e *ErrFunctionNotFound) Error() string {
	return fmt.Sprintf("function %s not found", e.Name)
}

// ErrAmbiguousFunction is returned when a bare function name matches functions in more than one collection
type ErrAmbiguousFunction struct {
	Name       string
	Candidates []string
}

func (e *ErrAmbiguousFunction) Error() string {
	return fmt.Sprintf("function %s is ambiguous, use one of: %s", e.Name, strings.Join(e.Candidates, ", "))
}

func (f ExternalFunctions) String() string {
	vals := make([]string, 0)
	for _, v := range f {
		vals = append(vals, v.String())
	}
	slices.Sort(vals)
	return fmt.Sprintf("%v", vals)
}

// Resolve finds a function by name. The name can be:
// * an exact map key
// * a qualified name (collection:name)
// * a prefixed name (collection_name) or a bare name, as long as only one function matches it. If more functions
// match, an ErrAmbiguousFunction is returned: no form takes precedence over another.
func (f ExternalFunctions) Resolve(name string) (ExternalFunction, error) {
	if fx, ok := f[name]; ok {
		return withDefaultName(name, fx), nil
	}
	matches := make([]ExternalFunction, 0)
	for k, v := range f {
		v = withDefaultName(k, v)
		if v.QualifiedName() == name {
			return v, nil
		}
		if v.matchesShortName(name) {
			matches = append(matches, v)
		}
	}
	switch len(matches) {
	case 0:
		return ExternalFunction{}, &ErrFunctionNotFound{Name: name}
	case 1:
		return matches[0], nil
	}
	candidates := make([]string, 0, len(matches))
	for _, m := range matches {
		candidates = append(candidates, m.QualifiedName())
	}
	slices.Sort(candidates)
	return ExternalFunction{}, &ErrAmbiguousFunction{Name: name, Candidates: candidates}
}

// Get returns a function by name (see Resolve). If the function cannot be resolved, a zero value is returned.
func (f ExternalFunctions) Get(name string) ExternalFunction {
	fx, _ := f.Resolve(name)
	return fx
}

// ListByCollection returns a subset of functions, filtered by MCP server or collection
func (f ExternalFunctions) ListByCollection(collection string) ExternalFunctions {
	out := ExternalFunctions{}
	for k, v := range f {
		if v.Collection == collection {
			out[k] = withDefaultName(k, v)
		}
	}
	return out
}

// Collisions returns the bare or prefixed function names that match more than one function, mapped to the
// qualified names of the functions they match.
func (f ExternalFunctions) Collisions() map[string][]string {
	byName := make(map[string][]string)
	for k, v := range f {
		v = withDefaultName(k, v)
		for _, n := range slices.Compact([]string{v.Name, v.PrefixedName()}) {
			byName[n] = append(byName[n], v.QualifiedName())
		}
	}
	out := make(map[string][]string)
	for name, qualified := range byName {
		if len(qualified) > 1 {
			slices.Sort(qualified)
			out[name] = qualified
		}
	}
	return out
}

// WithFunctions returns a new set of functions, merging the current ones with the provided ones. All functions are
// keyed by their qualified name, so functions with the same name in different collections don't overwrite each other.
func (f ExternalFunctions) WithFunctions(functions ExternalFunctions) ExternalFunctions {
	dst := make(ExternalFunctions, len(f)+len(functions))
	for _, src := range []ExternalFunctions{f, functions} {
		for k, v := range src {
			v = withDefaultName(k, v)
			dst[v.QualifiedName()] = v
		}
	}
	return dst
}

// withDefaultName fills the function name with the map key, if the name is missing
func withDefaultName(key string, f ExternalFunction) ExternalFunction {
	if f.Name == "" {
		f.Name = key
	}
	return f
}
