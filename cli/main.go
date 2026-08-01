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
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// resolveConfigPath finds the first existing configuration file.
// If none are found, it returns the path where the configuration file should be created.
func resolveConfigPath() (string, bool) {
	// 1. Check environment variable override
	if envPath := os.Getenv("FRAGS_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, true
		}
	}

	// 2. Check current working directory
	cwdPath := ".env"
	if _, err := os.Stat(cwdPath); err == nil {
		return cwdPath, true
	}

	// 3. Check standard User Config Directory
	var userConfigPath string
	if uConfigDir, err := os.UserConfigDir(); err == nil {
		userConfigPath = filepath.Join(uConfigDir, "frags", ".env")
		if _, err := os.Stat(userConfigPath); err == nil {
			return userConfigPath, true
		}
	}

	// 4. Check Application Executable Directory
	var exeConfigPath string
	if exePath, err := os.Executable(); err == nil {
		exeConfigPath = filepath.Join(filepath.Dir(exePath), ".env")
		if _, err := os.Stat(exeConfigPath); err == nil {
			return exeConfigPath, true
		}
	}

	// 5. If none exist, return the default location to write a new one:
	// User Configuration Directory to prevent current directory pollution.
	if uConfigDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(uConfigDir, "frags", ".env"), false
	}
	return cwdPath, false
}

func main() {
	configPath, exists := resolveConfigPath()

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if !exists {
		fmt.Println("No configuration file found. Starting interactive configuration editor...")
		if err := runInteractiveConfig(""); err != nil {
			fmt.Printf("Error starting config editor: %v\n", err)
		}
		return
	}

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("error reading config file: %v", err))
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	_ = rootCmd.Execute()
}
