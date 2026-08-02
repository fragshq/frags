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
	"reflect"
	"strconv"
	"strings"

	"github.com/theirish81/tui"
)

type EnvLine struct {
	Raw   string
	Key   string
	Value string
}

func parseEnvFile(path string) ([]EnvLine, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	var envLines []EnvLine
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			envLines = append(envLines, EnvLine{Raw: line})
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			envLines = append(envLines, EnvLine{Raw: line})
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// strip quotes if any
		if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
			(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
			val = val[1 : len(val)-1]
		}
		envLines = append(envLines, EnvLine{Key: key, Value: val, Raw: line})
	}
	return envLines, nil
}

func writeEnvFile(path string, envLines []EnvLine) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	var lines []string
	for _, el := range envLines {
		if el.Key != "" {
			lines = append(lines, fmt.Sprintf(`%s="%s"`, el.Key, el.Value))
		} else {
			lines = append(lines, el.Raw)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// defaultEnvSettings defines the initial keys and their default values used to partly populate a brand new env/config file.
// You can manually fill or adjust these default values here.
var defaultEnvSettings = []struct {
	Key   string
	Value string
}{
	{Key: "AI_ENGINE", Value: "gemini"},
	{Key: "MODEL", Value: "gemini-3.1-flash-lite"},
	{Key: "PARALLEL_WORKERS", Value: "1"},
	{Key: "OLLAMA_BASE_URL", Value: "http://localhost:11434"},
	{Key: "GEMINI_LOCATION", Value: "global"},
	{Key: "GEMINI_PROJECT_ID", Value: ""},
	{Key: "GEMINI_SERVICE_ACCOUNT_PATH", Value: ""},
	{Key: "CHATGPT_API_KEY", Value: ""},
	{Key: "CHATGPT_BASE_URL", Value: "https://api.openai.com/v1"},
	{Key: "ANTHROPIC_API_KEY", Value: ""},
	{Key: "THINKING_LEVEL", Value: "LOW"},
	{Key: "NUM_PREDICT", Value: "6400"},
	{Key: "TEMPERATURE", Value: "0.2"},
	{Key: "TOP_K", Value: "64"},
	{Key: "TOP_P", Value: "0.95"},
}

func createDefaultEnvLines() []EnvLine {
	var lines []EnvLine
	lines = append(lines, EnvLine{Raw: "# Frags CLI Configuration"})
	lines = append(lines, EnvLine{Raw: ""})
	for _, setting := range defaultEnvSettings {
		lines = append(lines, EnvLine{
			Key:   setting.Key,
			Value: setting.Value,
		})
	}
	return lines
}

func envLinesToConfig(envLines []EnvLine) Config {
	var c Config
	val := reflect.ValueOf(&c).Elem()
	typ := val.Type()

	envMap := make(map[string]string)
	for _, el := range envLines {
		if el.Key != "" {
			envMap[el.Key] = el.Value
		}
	}

	for i := 0; i < val.NumField(); i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}
		strVal, ok := envMap[tag]
		if !ok || strVal == "" {
			continue
		}
		fieldVal := val.Field(i)
		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(strVal)
		case reflect.Int:
			if iv, err := strconv.Atoi(strVal); err == nil {
				fieldVal.SetInt(int64(iv))
			}
		case reflect.Float32:
			if fv, err := strconv.ParseFloat(strVal, 32); err == nil {
				fieldVal.SetFloat(fv)
			}
		case reflect.Bool:
			if bv, err := strconv.ParseBool(strVal); err == nil {
				fieldVal.SetBool(bv)
			}
		}
	}
	return c
}

func configToEnvLines(envLines []EnvLine, c Config) []EnvLine {
	val := reflect.ValueOf(c)
	typ := val.Type()

	updates := make(map[string]string)
	for i := 0; i < val.NumField(); i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}
		fieldVal := val.Field(i)
		var strVal string
		switch fieldVal.Kind() {
		case reflect.String:
			strVal = fieldVal.String()
		case reflect.Int:
			strVal = strconv.Itoa(int(fieldVal.Int()))
		case reflect.Float32:
			strVal = strconv.FormatFloat(fieldVal.Float(), 'g', -1, 32)
		case reflect.Float64:
			strVal = strconv.FormatFloat(fieldVal.Float(), 'g', -1, 64)
		case reflect.Bool:
			strVal = strconv.FormatBool(fieldVal.Bool())
		}
		updates[tag] = strVal
	}

	updatedMap := make(map[string]bool)
	for i, el := range envLines {
		if el.Key != "" {
			if newVal, ok := updates[el.Key]; ok {
				envLines[i].Value = newVal
				updatedMap[el.Key] = true
			}
		}
	}

	for k, v := range updates {
		if !updatedMap[k] && v != "" {
			envLines = append(envLines, EnvLine{Key: k, Value: v})
		}
	}

	return envLines
}

func runInteractiveConfig(preselectedPath string) error {
	filePath := preselectedPath
	if filePath == "" {
		var fileOpts []fileOption
		fileOpts = append(fileOpts, fileOption{
			label: "Current Directory (.env)",
			path:  ".env",
		})
		if uConfigDir, err := os.UserConfigDir(); err == nil {
			fileOpts = append(fileOpts, fileOption{
				label: "User Config Directory (global)",
				path:  filepath.Join(uConfigDir, "frags", ".env"),
			})
		}

		selected, err := ChooseFile("Select which configuration file to edit:", fileOpts)
		if err != nil {
			return err
		}
		filePath = selected
	}

	lines, err := parseEnvFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			lines = createDefaultEnvLines()
		} else {
			return err
		}
	}

	cfgStruct := envLinesToConfig(lines)

	err = tui.Run[Config](&cfgStruct, tui.WithTitle[Config]("Frags Config Editor"))
	if err != nil {
		if errors.Is(err, tui.ErrCancelled) {
			fmt.Println("\nEditing cancelled. No changes were saved.")
			return nil
		}
		return err
	}

	updatedLines := configToEnvLines(lines, cfgStruct)
	err = writeEnvFile(filePath, updatedLines)
	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully saved configuration to: %s\n", filePath)
	return nil
}
