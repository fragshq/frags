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

var collectionsCmd = &cobra.Command{
	Use:   "collections",
	Short: "Manage tools collections",
	Long:  `Manage tools collections. Use "edit" to edit collections configurations interactively.`,
}

var collectionsViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Print the current collections configuration",
	Long:  "Print the current collections configuration",
	Run: func(cmd *cobra.Command, args []string) {
		tools, err := readToolsFile()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		dataBytes, err := json.MarshalIndent(tools.Collections, "", " ")
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		out, _ := highlightOutput(dataBytes, formatJSON)
		fmt.Println(string(out))
	},
}

var collectionsEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit collections interactively",
	Long:  `Launches a terminal UI to view, edit, add, and remove tools collections in project or global environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInteractiveCollections(""); err != nil {
			cmd.PrintErrln("Error:", err)
		}
	},
}

func init() {
	collectionsCmd.AddCommand(collectionsViewCmd)
	collectionsCmd.AddCommand(collectionsEditCmd)
}
