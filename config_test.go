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

	"github.com/stretchr/testify/assert"
)

func TestMcpServerConfigs_AsToolDefinitions(t *testing.T) {
	cfg := McpServerConfigs{
		"foo": McpServerConfig{},
		"bar": McpServerConfig{},
	}
	defs := cfg.AsToolDefinitions()
	assert.Len(t, defs, 2)
	for _, def := range defs {
		assert.Contains(t, []string{"foo", "bar"}, def.Name)
		assert.Equal(t, ToolTypeMCP, def.Type)
	}
}

func TestToolsCollectionConfigs_AsToolDefinitions(t *testing.T) {
	cfg := ToolsCollectionConfigs{
		"foo": CollectionConfig{},
		"bar": CollectionConfig{},
	}
	defs := cfg.AsToolDefinitions()
	assert.Len(t, defs, 2)
	for _, def := range defs {
		assert.Contains(t, []string{"foo", "bar"}, def.Name)
		assert.Equal(t, ToolTypeCollection, def.Type)
	}
}

func TestMcpServerConfigs_AsToolDefinitions_SkipsDisabled(t *testing.T) {
	cfg := McpServerConfigs{
		"foo": McpServerConfig{},
		"bar": McpServerConfig{Disabled: true},
	}
	defs := cfg.AsToolDefinitions()
	assert.Len(t, defs, 1)
	assert.Equal(t, "foo", defs[0].Name)
}

func TestToolsCollectionConfigs_AsToolDefinitions_SkipsDisabled(t *testing.T) {
	cfg := ToolsCollectionConfigs{
		"foo": CollectionConfig{},
		"bar": CollectionConfig{Disabled: true},
	}
	defs := cfg.AsToolDefinitions()
	assert.Len(t, defs, 1)
	assert.Equal(t, "foo", defs[0].Name)
}

// TestRequiredTools_Check_Disabled makes sure a plan requiring a disabled tool fails the check instead of running
// without it
func TestRequiredTools_Check_Disabled(t *testing.T) {
	cfg := ToolsConfig{
		McpServers:  McpServerConfigs{"s1": {Disabled: true}},
		Collections: ToolsCollectionConfigs{"http": {ToolType: "http", Disabled: true}},
	}
	defs := cfg.AsToolDefinitions()
	assert.Error(t, RequiredTools{{Name: "s1", Type: ToolTypeMCP}}.Check(defs))
	assert.Error(t, RequiredTools{{Name: "http", Type: ToolTypeCollection}}.Check(defs))
}
