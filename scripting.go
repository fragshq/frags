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

package frags

import (
	"encoding/json"
	"fmt"

	"github.com/theirish81/frags/util"
)

type ScriptType string

const Code ScriptType = "code"
const Kbs ScriptType = "kbs"

type ScriptComponent struct {
	Type        ScriptType `yaml:"type" json:"type"`
	Script      string     `yaml:"script" json:"script"`
	Description string     `yaml:"description" json:"description,omitempty"`
	Parameters  Parameters `yaml:"parameters" json:"parameters,omitempty"`
}

type ScriptComponents map[string]ScriptComponent

func (s ScriptComponents) Find(name string, typ ScriptType) (*ScriptComponent, bool) {
	if c, ok := s[name]; ok {
		if c.Type == typ {
			return &c, true
		}
	}
	return nil, false
}

func (s ScriptComponents) ListByType(typ ScriptType) ScriptComponents {
	out := ScriptComponents{}
	for k, v := range s {
		if v.Type == typ {
			out[k] = v
		}
	}
	return out
}

func (s ScriptComponents) Describe() string {
	data := ""
	for k, v := range s {
		data += fmt.Sprintf("Script name: %s\n", k)
		if len(v.Description) > 0 {
			data += fmt.Sprintf("Description: %s\n", v.Description)
		}
		if v.Parameters != nil && len(v.Parameters) > 0 {
			data += "Arguments:\n"
			for _, a := range v.Parameters {
				data += fmt.Sprintf("Name: %s\n", a.Name)
				sBytes, _ := json.Marshal(a.Schema)
				data += fmt.Sprintf("Schema:\n%s\n===\n", string(sBytes))
			}
		}
	}
	return data
}

// ScriptEngine is the interface that wraps the RunCode method. Frags provides NO script engines, it's the program
// that includes Frags that provides one, if necessary. Beware though, most script engines pose a security risk.
type ScriptEngine interface {
	RunCode(ctx *util.FragsContext, code string, params any, runner ExportableRunner) (any, error)
}

type DummyScriptEngine struct{}

func (d *DummyScriptEngine) RunCode(_ *util.FragsContext, _ string, _ any, _ ExportableRunner) (any, error) {
	return make(map[string]any), nil
}

type KbsEngine interface {
	Run(ctx *util.FragsContext, code string, query string, runner ExportableRunner) (any, error)
}

type DummyKbsEngine struct{}

func (d *DummyKbsEngine) Run(ctx *util.FragsContext, code string, query string, runner ExportableRunner) (any, error) {
	return make([]map[string]any, 0), nil
}
