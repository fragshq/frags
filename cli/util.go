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
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	fmlCompiler "github.com/fragshq/fml/compiler"
	fmlParser "github.com/fragshq/fml/parser"
	"github.com/fragshq/frags"
	"github.com/fragshq/frags/util"
	"gopkg.in/yaml.v3"
)

// sliceToMap converts a slice of strings with the key=value format into a map of strings. If ignoreErrors is true,
// strings that do not conform to the format are ignored
func sliceToMap(s []string, ignoreErrors bool) (map[string]any, error) {
	m := make(map[string]any, len(s))
	for _, v := range s {
		if matched, _ := regexp.Match("^[^=]+=[^=]+$", []byte(v)); matched {
			kv := strings.SplitN(v, "=", 2)
			m[kv[0]] = kv[1]
		} else if !ignoreErrors {
			return m, errors.New("invalid parameter format: " + v)
		}

	}
	return m, nil
}

func printDebugAny(res any) {
	switch reflect.ValueOf(res).Kind() {
	case reflect.Map, reflect.Slice:
		fmt.Println(util.MustJsonIndentString(res))
	default:
		fmt.Printf("%v", res)
	}
}

func parsePlan(filename string, data []byte) (frags.SessionManager, error) {
	sm := frags.NewSessionManager()
	if filepath.Ext(filename) == ".fml" {
		parser, _ := fmlParser.NewParser()
		parsedFml, err := parser.ParseString(filename, string(data))
		if err != nil {
			return sm, err
		}
		planYaml, err := fmlCompiler.New(parsedFml).Compile()
		if err != nil {
			return sm, err
		}
		if data, err = yaml.Marshal(planYaml); err != nil {
			return sm, err
		}
	}
	err := sm.FromYAML(data)
	return sm, err
}
