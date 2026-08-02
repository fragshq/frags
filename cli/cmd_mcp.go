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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long:  `Manage MCP servers. Use "edit" to edit MCP configurations interactively.`,
}

var mcpViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Print the current MCP configuration",
	Long:  "Print the current MCP configuration",
	Run: func(cmd *cobra.Command, args []string) {
		tools, err := readToolsFile()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		dataBytes, err := json.MarshalIndent(tools.McpServers, "", " ")
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		out, _ := highlightOutput(dataBytes, formatJSON)
		fmt.Println(string(out))
	},
}

var mcpEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit MCP servers interactively",
	Long:  `Launches a terminal UI to view, edit, add, and remove MCP servers in project or global environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInteractiveMcp(""); err != nil {
			cmd.PrintErrln("Error:", err)
		}
	},
}

func init() {
	mcpCmd.AddCommand(mcpEditCmd)
	mcpCmd.AddCommand(mcpViewCmd)
}
