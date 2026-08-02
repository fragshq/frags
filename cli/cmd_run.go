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
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/util"
)

var runCmd = &cobra.Command{
	Use:   "run <path/to/plan.yaml|.fml>",
	Short: "Run a frags plan from a YAML or FML file.",
	Long:  `Run a frags plan from a YAML or FML file. This is frags CLI core functionality.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate flags and input
		if err := validateRunArgs(args); err != nil {
			cmd.PrintErrln(err)
			return
		}

		planData, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		sm, err := parsePlan(args[0], planData)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		toolsConfig, err := readToolsFile()
		if err != nil {
			cmd.PrintErrln(err)
		}
		paramsMap, err := sliceToMap(params, false)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		ctx := util.WithFragsContext(cmd.Context(), 15*time.Minute)
		defer ctx.Cancel(nil)

		var result util.ProgMap

		useInteractive := !plain && !debug && isatty.IsTerminal(os.Stderr.Fd())

		if useInteractive {
			result, err = runWithBubbleTea(ctx, sm, paramsMap, toolsConfig, args)
		} else {
			var streamerLogger *log.StreamerLogger
			if debug {
				streamerLogger = log.NewStreamerLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				})), nil, log.DebugChannelLevel)
			} else {
				streamerLogger = log.NewStreamerLogger(slog.Default(), nil, log.InfoChannelLevel)
			}
			result, err = execute(ctx, sm, paramsMap, toolsConfig,
				resources.NewFileResourceLoader(filepath.Dir(args[0])), streamerLogger)
		}
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// render output according to the chosen format
		text, err := renderResult(result)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// write to file or stdout
		if output != "" {
			if err := os.WriteFile(output, text, 0o644); err != nil {
				cmd.PrintErrln(err)
			}
			return
		}

		useColor := output == "" && !plain && !debug && isatty.IsTerminal(os.Stdout.Fd())
		if useColor {
			if highlighted, err := highlightOutput(text, format); err == nil {
				text = highlighted
			}
		}

		fmt.Print(string(text))
	},
}

var plain bool

func init() {
	runCmd.Flags().StringVarP(&format, "format", "f", formatYAML, "output format (yaml, json or template)")
	runCmd.Flags().StringVarP(&output, "output", "o", "", "output file")
	runCmd.Flags().StringVarP(&templatePath, "template", "t", "", "go template file (used with -f template)")
	runCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "a parameter to pass to the plan (can be specified multiple times)")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	runCmd.Flags().BoolVar(&plain, "plain", false, "enable plain output mode (use standard logger)")
}

// validateRunArgs checks basic flag constraints and file existence.
func validateRunArgs(args []string) error {
	if format == formatTemplate && templatePath == "" {
		return fmt.Errorf("template path must be specified when using format=template")
	}
	if _, err := os.Stat(args[0]); err != nil {
		return fmt.Errorf("input file error: %w", err)
	}
	if format != formatYAML && format != formatJSON && format != formatTemplate {
		return fmt.Errorf("unsupported format %q", format)
	}
	return nil
}

func highlightOutput(text []byte, formatType string) ([]byte, error) {
	var lexer string
	switch formatType {
	case formatJSON:
		lexer = "json"
	case formatTemplate:
		lexer = "markdown"
	case formatEnv:
		lexer = "bash"
	default:
		lexer = "yaml"
	}

	style := "monokai"
	if !lipgloss.HasDarkBackground() {
		style = "github"
	}

	var buf bytes.Buffer
	err := quick.Highlight(&buf, string(text), lexer, "terminal256", style)
	if err != nil {
		return text, err
	}
	return buf.Bytes(), nil
}
