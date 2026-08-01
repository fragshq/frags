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
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type editState int

const (
	stateChooseFile editState = iota
	stateListKeys
	stateEditValue
)

type configKey struct {
	envKey      string
	label       string
	description string
	defVal      string
}

var standardKeys = []configKey{
	{"AI_ENGINE", "AI Engine", "AI engine to use (gemini, ollama, chatgpt, anthropic, dummy)", "gemini"},
	{"MODEL", "Model Name", "The specific model to use (e.g., gemini-2.5-flash, qwen3:latest)", ""},
	{"TEMPERATURE", "Temperature", "Model-specific temperature parameter (e.g., 0.3)", "0.3"},
	{"TOP_K", "Top K", "Model-specific top_k parameter (e.g., 64)", "64"},
	{"TOP_P", "Top P", "Model-specific top_p parameter (e.g., 0.95)", "0.95"},
	{"NUM_PREDICT", "Num Predict", "The maximum number of tokens to predict", "1024"},
	{"PARALLEL_WORKERS", "Parallel Workers", "The number of parallel workers to use for processing", "1"},
	{"OLLAMA_BASE_URL", "Ollama Base URL", "The base URL for your Ollama instance", "http://localhost:11434"},
	{"GEMINI_SERVICE_ACCOUNT_PATH", "Gemini SA Path", "Path to Google Cloud service account JSON file", ""},
	{"GEMINI_PROJECT_ID", "Gemini Project ID", "Your Google Cloud project ID", ""},
	{"GEMINI_LOCATION", "Gemini Location", "The Google Cloud region for Gemini API", "global"},
	{"CHATGPT_API_KEY", "ChatGPT API Key", "Your OpenAI API key", ""},
	{"CHATGPT_BASE_URL", "ChatGPT Base URL", "The base URL for ChatGPT API", "https://api.openai.com/v1"},
	{"ANTHROPIC_API_KEY", "Anthropic API Key", "Your Anthropic API key", ""},
	{"THINKING_LEVEL", "Thinking Level", "Thinking level for supported models (LOW, MEDIUM, HIGH)", "LOW"},
	{"OAUTH_DISABLED", "OAuth Disabled", "Whether OAuth authentication is disabled (true, false)", "false"},
}

type EnvLine struct {
	Raw   string
	Key   string
	Value string
}

type fileOption struct {
	label string
	path  string
}

type interactiveModel struct {
	state       editState
	fileOptions []fileOption
	fileCursor  int

	filePath string
	envLines []EnvLine
	values   map[string]string // KEY -> VALUE

	listCursor    int
	listScroll    int
	input         textinput.Model
	enumCursor    int
	editingCustom bool

	saved bool
	err   error
}

func (m *interactiveModel) getEnumOptions(sk configKey) []string {
	if sk.envKey == "AI_ENGINE" {
		return []string{"anthropic", "chatgpt", "gemini", "ollama"}
	}
	if sk.envKey == "MODEL" {
		engine := strings.ToLower(m.values["AI_ENGINE"])
		var opts []string
		switch engine {
		case "gemini":
			opts = []string{"gemini-3.1-pro-preview", "gemini-3.5-flash-lite", "gemini-3.5-flash"}
		case "anthropic":
			opts = []string{"claude-sonnet-5", "claude-opus-5", "claude-haiku-4-5"}
		case "chatgpt":
			opts = []string{"gpt-4o-mini", "gpt-4o"}
		}
		if len(opts) > 0 {
			opts = append(opts, "Custom (type manually)...")
			return opts
		}
	}
	return nil
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
	// Ensure directory exists
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

func createDefaultEnvLines() []EnvLine {
	var lines []EnvLine
	lines = append(lines, EnvLine{Raw: "# Frags CLI Configuration"})
	lines = append(lines, EnvLine{Raw: ""})
	for _, sk := range standardKeys {
		lines = append(lines, EnvLine{
			Key:   sk.envKey,
			Value: sk.defVal,
			Raw:   fmt.Sprintf(`%s="%s"`, sk.envKey, sk.defVal),
		})
	}
	return lines
}

func newInteractiveModel() interactiveModel {
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

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = 40

	return interactiveModel{
		state:       stateChooseFile,
		fileOptions: fileOpts,
		input:       ti,
		values:      make(map[string]string),
	}
}

func (m interactiveModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.state {
		case stateChooseFile:
			switch msg.String() {
			case "up", "k":
				if m.fileCursor > 0 {
					m.fileCursor--
				}
			case "down", "j":
				if m.fileCursor < len(m.fileOptions)-1 {
					m.fileCursor++
				}
			case "enter":
				m.filePath = m.fileOptions[m.fileCursor].path
				// Load file if exists, else load empty default lines
				lines, err := parseEnvFile(m.filePath)
				if err != nil {
					if os.IsNotExist(err) {
						m.envLines = createDefaultEnvLines()
					} else {
						m.err = err
						return m, tea.Quit
					}
				} else {
					m.envLines = lines
				}

				// Populate values map
				for _, el := range m.envLines {
					if el.Key != "" {
						m.values[el.Key] = el.Value
					}
				}
				// Also populate standard keys with empty if they aren't in the file yet
				for _, sk := range standardKeys {
					if _, ok := m.values[sk.envKey]; !ok {
						m.values[sk.envKey] = ""
					}
				}

				m.state = stateListKeys
				m.listCursor = 0
				m.listScroll = 0
			case "q", "esc":
				return m, tea.Quit
			}

		case stateListKeys:
			switch msg.String() {
			case "up", "k":
				if m.listCursor > 0 {
					m.listCursor--
					if m.listCursor < m.listScroll {
						m.listScroll = m.listCursor
					}
				}
			case "down", "j":
				if m.listCursor < len(standardKeys)-1 {
					m.listCursor++
					if m.listCursor >= m.listScroll+10 {
						m.listScroll = m.listCursor - 9
					}
				}
			case "enter":
				m.state = stateEditValue
				m.editingCustom = false
				sk := standardKeys[m.listCursor]
				opts := m.getEnumOptions(sk)
				if len(opts) > 0 {
					currentVal := m.values[sk.envKey]
					m.enumCursor = 0
					for idx, opt := range opts {
						if opt == currentVal {
							m.enumCursor = idx
							break
						}
					}
				} else {
					m.input.SetValue(m.values[sk.envKey])
					m.input.Focus()
				}
			case "u":
				// Unset/remove the selected key
				sk := standardKeys[m.listCursor]
				m.values[sk.envKey] = ""
				var newLines []EnvLine
				for _, el := range m.envLines {
					if el.Key != sk.envKey {
						newLines = append(newLines, el)
					}
				}
				m.envLines = newLines
			case "s":
				// Save & Quit
				err := writeEnvFile(m.filePath, m.envLines)
				if err != nil {
					m.err = err
				} else {
					m.saved = true
				}
				return m, tea.Quit
			case "q", "esc":
				m.state = stateChooseFile
			}

		case stateEditValue:
			sk := standardKeys[m.listCursor]
			opts := m.getEnumOptions(sk)
			if len(opts) > 0 && !m.editingCustom {
				switch msg.String() {
				case "up", "k":
					if m.enumCursor > 0 {
						m.enumCursor--
					}
				case "down", "j":
					if m.enumCursor < len(opts)-1 {
						m.enumCursor++
					}
				case "enter":
					selected := opts[m.enumCursor]
					if selected == "Custom (type manually)..." {
						m.editingCustom = true
						m.input.SetValue(m.values[sk.envKey])
						m.input.Focus()
					} else {
						newVal := selected
						m.values[sk.envKey] = newVal

						found := false
						for i, el := range m.envLines {
							if el.Key == sk.envKey {
								m.envLines[i].Value = newVal
								found = true
								break
							}
						}
						if !found {
							m.envLines = append(m.envLines, EnvLine{
								Key:   sk.envKey,
								Value: newVal,
							})
						}
						m.state = stateListKeys
					}
				case "esc":
					m.state = stateListKeys
				}
			} else {
				switch msg.String() {
				case "enter":
					newVal := m.input.Value()
					m.values[sk.envKey] = newVal

					// Update or append in envLines
					found := false
					for i, el := range m.envLines {
						if el.Key == sk.envKey {
							m.envLines[i].Value = newVal
							found = true
							break
						}
					}
					if !found {
						m.envLines = append(m.envLines, EnvLine{
							Key:   sk.envKey,
							Value: newVal,
						})
					}

					m.state = stateListKeys
				case "esc":
					if len(opts) > 0 {
						m.editingCustom = false
					} else {
						m.state = stateListKeys
					}
				default:
					var cmd tea.Cmd
					m.input, cmd = m.input.Update(msg)
					return m, cmd
				}
			}
		}
	}

	return m, nil
}

func (m interactiveModel) View() string {
	var s strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 2).
		MarginBottom(1)

	s.WriteString("\n" + headerStyle.Render("⚡ FRAGS CONFIG EDITOR ⚡") + "\n\n")

	switch m.state {
	case stateChooseFile:
		s.WriteString("  Select which configuration file to edit:\n\n")
		for i, opt := range m.fileOptions {
			cursor := "  "
			labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BCBCBC"))
			if i == m.fileCursor {
				cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF007F")).Render("> ")
				labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF007F"))
			}

			// check if file exists
			existsStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87")).Render("(exists)")
			if _, err := os.Stat(opt.path); os.IsNotExist(err) {
				existsStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Render("(will create)")
			}

			s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, labelStyle.Render(opt.label), existsStr))
			s.WriteString(fmt.Sprintf("    %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#585858")).Render(opt.path)))
		}
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#767676")).Render("  (Use Up/Down arrows, Enter to select, q/esc to quit)") + "\n")

	case stateListKeys:
		s.WriteString(fmt.Sprintf("  File: %s\n\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF87")).Render(m.filePath)))
		s.WriteString("  Select a configuration key to edit:\n\n")

		limit := m.listScroll + 10
		if limit > len(standardKeys) {
			limit = len(standardKeys)
		}

		for i := m.listScroll; i < limit; i++ {
			sk := standardKeys[i]
			cursor := "  "
			keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BCBCBC"))
			valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00AFFF"))

			if i == m.listCursor {
				cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF007F")).Render("> ")
				keyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF007F"))
				valStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF87"))
			}

			val := m.values[sk.envKey]
			if val == "" {
				val = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858")).Render("<not set>")
			} else {
				val = fmt.Sprintf(`"%s"`, val)
			}

			// Format columns cleanly
			s.WriteString(fmt.Sprintf("%s%-30s = %s\n", cursor, keyStyle.Render(sk.envKey), valStyle.Render(val)))
		}

		// Scroll indicators
		scrollText := ""
		if m.listScroll > 0 {
			scrollText += "  ↑ (more keys above)"
		}
		if limit < len(standardKeys) {
			if scrollText != "" {
				scrollText += " | "
			}
			scrollText += "↓ (more keys below)"
		}
		if scrollText != "" {
			s.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#585858")).Render(scrollText) + "\n")
		}

		// Active key description box
		activeKey := standardKeys[m.listCursor]
		descStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Width(65).
			MarginTop(1)

		s.WriteString("\n" + descStyle.Render(
			fmt.Sprintf("%s\n%s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render(activeKey.label),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#BCBCBC")).Render(activeKey.description),
			),
		) + "\n")

		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#767676")).Render("\n  ↑/↓: Navigate | Enter: Edit | u: Unset | s: Save & Exit | q/esc: Go Back") + "\n")

	case stateEditValue:
		activeKey := standardKeys[m.listCursor]
		s.WriteString(fmt.Sprintf("  Editing: %s\n\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF007F")).Render(activeKey.envKey)))
		s.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#BCBCBC")).Render(activeKey.description)))

		opts := m.getEnumOptions(activeKey)
		if len(opts) > 0 && !m.editingCustom {
			s.WriteString("  Choose an option:\n\n")
			for idx, opt := range opts {
				cursor := "  "
				optStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BCBCBC"))
				if idx == m.enumCursor {
					cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF007F")).Render("> ")
					if opt == "Custom (type manually)..." {
						optStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFAF00"))
					} else {
						optStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF87"))
					}
				}
				s.WriteString(fmt.Sprintf("%s%s\n", cursor, optStyle.Render(opt)))
			}
			s.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#767676")).Render("  (Use Up/Down arrows, Enter to select, Esc to cancel)") + "\n")
		} else {
			s.WriteString("  Value:\n")
			s.WriteString("  " + m.input.View() + "\n\n")
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#767676")).Render("  (Enter to confirm, Esc to cancel)") + "\n")
		}
	}

	return s.String()
}

func runInteractiveConfig() error {
	m := newInteractiveModel()
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	mResult := finalModel.(interactiveModel)
	if mResult.err != nil {
		return mResult.err
	}
	if mResult.saved {
		fmt.Printf("\nSuccessfully saved configuration to: %s\n", mResult.filePath)
	}
	return nil
}
