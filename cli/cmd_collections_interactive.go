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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theirish81/frags"
	"github.com/theirish81/tui"
)

func runInteractiveCollections(preselectedPath string) error {
	filePath := preselectedPath
	if filePath == "" {
		var fileOpts []fileOption
		fileOpts = append(fileOpts, fileOption{
			label: "Current Directory (tools.json)",
			path:  "tools.json",
		})
		if uConfigDir, err := os.UserConfigDir(); err == nil {
			fileOpts = append(fileOpts, fileOption{
				label: "User Config Directory (global)",
				path:  filepath.Join(uConfigDir, "frags", "tools.json"),
			})
		}

		selected, err := ChooseFile("Select which tools configuration file to edit:", fileOpts)
		if err != nil {
			return err
		}
		filePath = selected
	}

	data, err := os.ReadFile(filePath)
	var config ExtendedToolsConfig
	if err != nil {
		if os.IsNotExist(err) {
			config = ExtendedToolsConfig{}
			config.McpServers = make(frags.McpServerConfigs)
			config.Collections = make(frags.ToolsCollectionConfigs)
		} else {
			return err
		}
	} else {
		config, err = parseToolsConfig(data)
		if err != nil {
			return err
		}
	}
	if config.Collections == nil {
		config.Collections = make(frags.ToolsCollectionConfigs)
	}

	err = tui.Run[frags.ToolsCollectionConfigs](&config.Collections,
		tui.WithTitle[frags.ToolsCollectionConfigs]("Frags Tools Collections Editor"),
	)
	if err != nil {
		if errors.Is(err, tui.ErrCancelled) {
			fmt.Println("\nEditing cancelled. No changes were saved.")
			return nil
		}
		return err
	}

	err = writeToolsFile(filePath, config)
	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully saved tools configuration to: %s\n", filePath)
	return nil
}
