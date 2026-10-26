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

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fragshq/doauth"
	"github.com/fragshq/frags"
	"github.com/fragshq/frags/httpfactory"
	"github.com/fragshq/frags/mcpauth"
	"github.com/spf13/cobra"
)

var (
	registerClientName         string
	registerInitialAccessToken string
	registerRegistrationURL    string
	registerForce              bool
)

var mcpRegisterCmd = &cobra.Command{
	Use:   "register <name> [url]",
	Short: "Register an OAuth client for an MCP server via Dynamic Client Registration",
	Long: `Registers an OAuth client with the authorization server protecting an MCP server (RFC 7591)
and stores the issued client_id and client_secret in the tools file.

If the server already exists in the tools file, the URL can be omitted and the existing one is used.
Otherwise a new server entry is created with the given URL.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		url := ""
		if len(args) > 1 {
			url = args[1]
		}
		if err := runMcpRegister(cmd.Context(), args[0], url); err != nil {
			cmd.PrintErrln("Error:", err)
		}
	},
}

func runMcpRegister(ctx context.Context, name string, url string) error {
	filePath := getToolsPath()
	config, err := readToolsFile()
	if err != nil {
		return err
	}
	if config.McpServers == nil {
		config.McpServers = make(frags.McpServerConfigs)
	}

	server, exists := config.McpServers[name]
	if url == "" {
		url = server.Url
	}
	if url == "" {
		return fmt.Errorf("no URL given and server %q has no URL in %s", name, filePath)
	}
	if exists && server.Url != "" && server.Url != url {
		return fmt.Errorf("server %q already exists with a different URL (%s)", name, server.Url)
	}
	if server.ClientID != nil && *server.ClientID != "" && !registerForce {
		return fmt.Errorf("server %q already has a client_id; use --force to register a new one", name)
	}
	server.Url = url

	// Custom headers configured for the server are also sent during discovery and registration.
	client, err := httpfactory.Instance.Builder().WithHeaders(server.HttpHeaders()).Build()
	if err != nil {
		return err
	}
	opts := []doauth.Option{
		doauth.WithHTTPClient(client),
		doauth.WithClientName(registerClientName),
	}
	if registerInitialAccessToken != "" {
		opts = append(opts, doauth.WithInitialAccessToken(registerInitialAccessToken))
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	reg, err := doauth.Register(ctx, doauth.Config{
		Name:            name,
		BaseURL:         url,
		RedirectURL:     mcpauth.DefaultRedirectURL,
		RegistrationURL: registerRegistrationURL,
	}, opts...)
	if err != nil {
		return err
	}

	// Public clients are not supported, so a registration without a secret is unusable.
	if reg.ClientSecret == "" {
		return fmt.Errorf("the authorization server registered %s as a public client (no client secret), which is not supported", reg.ClientID)
	}
	server.ClientID = &reg.ClientID
	server.ClientSecret = &reg.ClientSecret
	config.McpServers[name] = server

	if err := writeToolsFile(filePath, config); err != nil {
		return err
	}

	fmt.Printf("Registered client %s for %q", reg.ClientID, name)
	if reg.Issuer != "" {
		fmt.Printf(" with %s", reg.Issuer)
	}
	fmt.Printf("\nSaved to: %s\n", filePath)
	if reg.ClientSecretExpiresAt != 0 {
		fmt.Printf("Note: the client secret expires on %s\n", time.Unix(reg.ClientSecretExpiresAt, 0).Format(time.RFC3339))
	}
	return nil
}

func init() {
	mcpRegisterCmd.Flags().StringVar(&registerClientName, "client-name", "Frags", "client name shown on the consent screen")
	mcpRegisterCmd.Flags().StringVar(&registerInitialAccessToken, "initial-access-token", "", "bearer token for authorization servers that restrict registration")
	mcpRegisterCmd.Flags().StringVar(&registerRegistrationURL, "registration-url", "", "registration endpoint, skipping discovery")
	mcpRegisterCmd.Flags().BoolVar(&registerForce, "force", false, "replace an existing client_id")
}
