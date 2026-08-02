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
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the configuration",
	Long:  `Manage the configuration. Use "view" to print the current configuration or "edit" to edit it interactively.`,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Print the current configuration",
	Long:  `Prints the current configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		globalConfig, _ := yaml.Marshal(cfg)
		out, _ := highlightOutput(globalConfig, formatEnv)
		fmt.Println(string(out))
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the configuration interactively",
	Long:  `Launches a terminal UI to view and edit configuration keys for project or global environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInteractiveConfig(""); err != nil {
			cmd.PrintErrln("Error:", err)
		}
	},
}

func init() {
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configEditCmd)
}
