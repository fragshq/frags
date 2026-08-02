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
	"net/http"

	"github.com/theirish81/frags/mcpauth"
)

// ToolsConfig defines the configuration for the MCP clients and collections. This serves no specific purpose
// within Frags itself, but it can be used by integrating applications to standardize the configuration format.
type ToolsConfig struct {
	McpServers  McpServerConfigs       `json:"mcpServers,omitempty"`
	Collections ToolsCollectionConfigs `json:"collections,omitempty"`
}

// AsToolDefinitions returns the tools config as tool definitions
func (t ToolsConfig) AsToolDefinitions() ToolDefinitions {
	return append(t.McpServers.AsToolDefinitions(), t.Collections.AsToolDefinitions()...)
}

// CollectionConfig defines the configuration for a collection
type CollectionConfig struct {
	ToolType string            `json:"tool_type,omitempty" tui:"label=Tool Type,enum=fs|postgres|http,subtitle"`
	Params   map[string]string `json:"params,omitempty" tui:"label=Params"`
	Disabled bool              `json:"disabled" tui:"label=Disabled,!badge"`
}

// ToolsCollectionConfigs is a map of collection names to collection configurations
type ToolsCollectionConfigs map[string]CollectionConfig

func (t *ToolsCollectionConfigs) UnmarshalJSON(data []byte) error {
	// Unmarshal into a raw map first
	var raw map[string]CollectionConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*t = make(ToolsCollectionConfigs, len(raw))
	for key, config := range raw {
		if len(config.ToolType) == 0 {
			config.ToolType = key
		}
		(*t)[key] = config
	}
	return nil
}

// AsToolDefinitions returns the collection configs as tool definitions
func (t ToolsCollectionConfigs) AsToolDefinitions() ToolDefinitions {
	tools := ToolDefinitions{}
	for name, _ := range t {
		tools = append(tools, ToolDefinition{
			Name: name,
			Type: ToolTypeCollection,
		})
	}
	return tools
}

// McpServerConfig defines the configuration to connect to a MCP server
type McpServerConfig struct {
	Command          string            `json:"command,omitempty" tui:"label=Command,subtitle"`
	Args             []string          `json:"args,omitempty" tui:"label=Args"`
	Env              map[string]string `json:"env,omitempty" tui:"label=Env"`
	Cwd              string            `json:"cwd,omitempty" tui:"label=Cwd"`
	Transport        string            `json:"transport,omitempty" tui:"label=Transport"`
	Url              string            `json:"url,omitempty" tui:"label=URL,subtitle"`
	Headers          map[string]string `json:"headers,omitempty" tui:"label=Headers"`
	Disabled         bool              `json:"disabled" tui:"label=Disabled,!badge"`
	ClientID         *string           `json:"client_id,omitempty" tui:"label=Client ID"`
	ClientSecret     *string           `json:"client_secret,omitempty" tui:"label=Client Secret"`
	AuthorizationURL *string           `json:"authorization_url,omitempty" tui:"label=Authorization URL"`
	TokenURL         *string           `json:"token_url,omitempty" tui:"label=Token URL"`
	Token            *string           `json:"token,omitempty" tui:"label=Token"`
	// Placeholders for future functionalities and integrations
	PreAuthorizedOauth  *mcpauth.TokenResult `json:"pre_authorized_oauth,omitempty" tui:"label=Pre-Authorized OAuth"`
	AuthorizationMethod *string              `json:"authorization_method,omitempty" tui:"label=Authorization Method"`
}

func (m *McpServerConfig) HttpHeaders() http.Header {
	h := http.Header{}
	for k, v := range m.Headers {
		h.Add(k, v)
	}
	return h
}

// McpServerConfigs is a map of MCP servers
type McpServerConfigs map[string]McpServerConfig

// AsToolDefinitions returns the MCP server configs as tool definitions
func (m McpServerConfigs) AsToolDefinitions() ToolDefinitions {
	tools := ToolDefinitions{}
	for name, _ := range m {
		tools = append(tools, ToolDefinition{
			Name: name,
			Type: ToolTypeMCP,
		})
	}
	return tools
}
