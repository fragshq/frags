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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/theirish81/apicp"
	apiCpCollection "github.com/theirish81/apicp/collection"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/mcpauth"
	"github.com/theirish81/fragsfunctions/fs"
	"github.com/theirish81/fragsfunctions/http"
	"github.com/theirish81/fragsfunctions/postgres"
	"github.com/theirish81/sesat2"
)

var (
	toolsPath     string
	tokensPath    string
	pathsResolved bool
)

func resolvePaths() {
	if pathsResolved {
		return
	}
	// Check current directory first
	cwdTools := "tools.json"
	if _, err := os.Stat(cwdTools); err == nil {
		toolsPath = cwdTools
		tokensPath = "tokens.json"
		pathsResolved = true
		return
	}

	// Check user's config directory (cross platform)
	if uConfigDir, err := os.UserConfigDir(); err == nil {
		userTools := filepath.Join(uConfigDir, "frags", "tools.json")
		if _, err := os.Stat(userTools); err == nil {
			toolsPath = userTools
			tokensPath = filepath.Join(uConfigDir, "frags", "tokens.json")
			pathsResolved = true
			return
		}
	}

	// Default to current directory
	toolsPath = cwdTools
	tokensPath = "tokens.json"
	pathsResolved = true
}

func getToolsPath() string {
	resolvePaths()
	return toolsPath
}

func getTokensPath() string {
	resolvePaths()
	return tokensPath
}

type ExtendedToolsConfig struct {
	frags.ToolsConfig `json:",inline"`
	ApiCPs            map[string]ApiCPConfig `json:"apicps"`
}

type ApiCPConfig struct {
	Config   apicp.ApiCP `json:"config"`
	Disabled bool        `json:"disabled"`
}

// readToolsFile reads the tools configuration file and returns the parsed configuration
func readToolsFile() (ExtendedToolsConfig, error) {
	data, err := os.ReadFile(getToolsPath())
	if errors.Is(err, os.ErrNotExist) {
		data = []byte("{}")
	} else if err != nil {
		return ExtendedToolsConfig{}, err
	}
	return parseToolsConfig(data)
}

// parseToolsConfig parses the tools configuration file and returns the parsed configuration
func parseToolsConfig(data []byte) (ExtendedToolsConfig, error) {
	config := ExtendedToolsConfig{}
	err := json.Unmarshal(data, &config)
	return config, err
}

// connectMcpAndCollections connects to the MCP servers and returns the tools
func connectMcpAndCollections(ctx context.Context, toolsConfig ExtendedToolsConfig, logger *log.StreamerLogger) (frags.McpTools, []frags.ToolsCollection, frags.ToolDefinitions, frags.ExternalFunctions, error) {
	mcpTools := make(frags.McpTools, 0)
	toolCollections := make([]frags.ToolsCollection, 0)
	toolDefinitions := make(frags.ToolDefinitions, 0)
	functions := make(frags.ExternalFunctions, 0)
	toolDefinitions = toolsConfig.AsToolDefinitions()
	mcpTools = toolsConfig.McpServers.McpTools()
	if cfg.OauthDisabled {
		mcpTools.WithOAuthProvider(mcpauth.NewEmptyOauthProvider(true).WithCache(mcpauth.NewInMemoryCache()))
	} else {
		oauthCache, err := mcpauth.NewFsOauthCache(getTokensPath())
		if err != nil {
			return mcpTools, toolCollections, toolDefinitions, functions, err
		}
		mcpTools.WithOAuthProvider(mcpauth.NewEmptyOauthProvider(false).WithCache(oauthCache))
	}
	if err := mcpTools.Connect(ctx, logger); err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	functions, err := mcpTools.AsFunctions(ctx)
	if err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	for k, v := range toolsConfig.Collections {
		if v.Disabled {
			continue
		}
		switch v.ToolType {
		case "fs":
			t := fs.New()
			functions = functions.WithFunctions(t.AsFunctions())
			toolCollections = append(toolCollections, t)
		case "postgres":
			c, err := postgres.New(ctx, k, v.Params["postgres_url"])
			if err != nil {
				return mcpTools, toolCollections, toolDefinitions, functions, err
			}
			functions = functions.WithFunctions(c.AsFunctions())
			toolCollections = append(toolCollections, c)
		case "http":
			client, _ := sesat2.New().Build()
			c := http.New(k, nil, client)
			functions = functions.WithFunctions(c.AsFunctions())
			toolCollections = append(toolCollections, c)
		}
	}
	for k, v := range toolsConfig.ApiCPs {
		v.Config.Logger(slog.Default())
		client, _ := sesat2.New().Build()
		c, err := apiCpCollection.New(k, &v.Config, client)
		if err != nil {
			return mcpTools, toolCollections, toolDefinitions, functions, err
		}
		functions = functions.WithFunctions(c.AsFunctions())
		toolCollections = append(toolCollections, c)

		toolDefinitions = append(toolDefinitions, frags.ToolDefinition{
			Name: c.Name(),
			Type: "aicp",
		})
	}
	return mcpTools, toolCollections, toolDefinitions, functions, nil
}
