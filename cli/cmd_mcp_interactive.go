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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theirish81/frags"
)

type mcpEditState int

const (
	stateMcpChooseFile mcpEditState = iota
	stateMcpListServers
	stateMcpEditServerFields
	stateMcpEditFieldValue
)

type mcpFieldKey string

const (
	fieldMcpName         mcpFieldKey = "Name"
	fieldMcpCommand      mcpFieldKey = "Command"
	fieldMcpArgs         mcpFieldKey = "Args"
	fieldMcpUrl          mcpFieldKey = "Url"
	fieldMcpDisabled     mcpFieldKey = "Disabled"
	fieldMcpClientID     mcpFieldKey = "ClientID"
	fieldMcpClientSecret mcpFieldKey = "ClientSecret"
	fieldMcpToken        mcpFieldKey = "Token"
	fieldMcpCwd          mcpFieldKey = "Cwd"
)

type mcpField struct {
	key         mcpFieldKey
	label       string
	description string
}

var mcpFields = []mcpField{
	{fieldMcpName, "Server Name", "Unique identifier for this MCP server"},
	{fieldMcpCommand, "Command", "Command to run local stdio server (e.g. node, python, npx)"},
	{fieldMcpArgs, "Args", "Space-separated arguments to pass to the command"},
	{fieldMcpUrl, "Url", "URL for SSE or WebSocket transport"},
	{fieldMcpDisabled, "Disabled", "Toggle whether this MCP server is disabled"},
	{fieldMcpClientID, "Client ID", "OAuth/API Client ID if needed"},
	{fieldMcpClientSecret, "Client Secret", "OAuth/API Client Secret if needed"},
	{fieldMcpToken, "Token", "Pre-configured bearer/auth token"},
	{fieldMcpCwd, "Cwd", "Working directory for the command execution"},
}

type interactiveMcpModel struct {
	state       mcpEditState
	fileOptions []fileOption
	fileCursor  int

	filePath string
	config   ExtendedToolsConfig
	servers  []string

	listCursor  int
	fieldCursor int
	input       textinput.Model

	activeServer string
	editingField mcpFieldKey

	saved bool
	err   error
}

func newInteractiveMcpModel(preselectedPath string) interactiveMcpModel {
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

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	m := interactiveMcpModel{
		state:       stateMcpChooseFile,
		fileOptions: fileOpts,
		input:       ti,
	}

	if preselectedPath != "" {
		m.filePath = preselectedPath
		err := m.loadConfig()
		if err != nil {
			m.err = err
			return m
		}
		m.state = stateMcpListServers
		m.listCursor = 0
	}

	return m
}

func (m *interactiveMcpModel) loadConfig() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.config = ExtendedToolsConfig{}
			m.config.McpServers = make(frags.McpServerConfigs)
			m.config.Collections = make(frags.ToolsCollectionConfigs)
			m.updateServerNames()
			return nil
		}
		return err
	}
	m.config, err = parseToolsConfig(data)
	if err != nil {
		return err
	}
	if m.config.McpServers == nil {
		m.config.McpServers = make(frags.McpServerConfigs)
	}
	m.updateServerNames()
	return nil
}

func (m *interactiveMcpModel) updateServerNames() {
	m.servers = make([]string, 0, len(m.config.McpServers))
	for name := range m.config.McpServers {
		m.servers = append(m.servers, name)
	}
	sort.Strings(m.servers)
}

func writeToolsFile(path string, config ExtendedToolsConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m interactiveMcpModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m interactiveMcpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.state {
		case stateMcpChooseFile:
			path, completed, quit := UpdateFileSelection(msg.String(), &m.fileCursor, m.fileOptions)
			if quit {
				return m, tea.Quit
			}
			if completed {
				m.filePath = path
				if err := m.loadConfig(); err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.state = stateMcpListServers
				m.listCursor = 0
			}

		case stateMcpListServers:
			switch msg.String() {
			case "up", "k":
				if m.listCursor > 0 {
					m.listCursor--
				}
			case "down", "j":
				if m.listCursor < len(m.servers)-1 {
					m.listCursor++
				}
			case "enter":
				if len(m.servers) > 0 {
					m.activeServer = m.servers[m.listCursor]
					m.state = stateMcpEditServerFields
					m.fieldCursor = 0
				}
			case "a":
				newName := "new-server"
				for i := 1; ; i++ {
					if _, exists := m.config.McpServers[newName]; !exists {
						break
					}
					newName = fmt.Sprintf("new-server-%d", i)
				}
				m.config.McpServers[newName] = frags.McpServerConfig{
					Disabled: false,
				}
				m.updateServerNames()
				m.activeServer = newName
				for i, s := range m.servers {
					if s == newName {
						m.listCursor = i
						break
					}
				}
				m.state = stateMcpEditServerFields
				m.fieldCursor = 0
			case "d":
				if len(m.servers) > 0 {
					delete(m.config.McpServers, m.servers[m.listCursor])
					m.updateServerNames()
					if m.listCursor >= len(m.servers) && m.listCursor > 0 {
						m.listCursor = len(m.servers) - 1
					}
				}
			case "s":
				if err := writeToolsFile(m.filePath, m.config); err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.saved = true
				return m, tea.Quit
			case "q", "esc":
				return m, tea.Quit
			}

		case stateMcpEditServerFields:
			switch msg.String() {
			case "up", "k":
				if m.fieldCursor > 0 {
					m.fieldCursor--
				}
			case "down", "j":
				if m.fieldCursor < len(mcpFields)-1 {
					m.fieldCursor++
				}
			case "enter":
				f := mcpFields[m.fieldCursor]
				cfg := m.config.McpServers[m.activeServer]
				if f.key == fieldMcpDisabled {
					cfg.Disabled = !cfg.Disabled
					m.config.McpServers[m.activeServer] = cfg
				} else {
					m.editingField = f.key
					m.state = stateMcpEditFieldValue
					var currentVal string
					switch f.key {
					case fieldMcpName:
						currentVal = m.activeServer
					case fieldMcpCommand:
						currentVal = cfg.Command
					case fieldMcpArgs:
						currentVal = strings.Join(cfg.Args, " ")
					case fieldMcpUrl:
						currentVal = cfg.Url
					case fieldMcpClientID:
						if cfg.ClientID != nil {
							currentVal = *cfg.ClientID
						}
					case fieldMcpClientSecret:
						if cfg.ClientSecret != nil {
							currentVal = *cfg.ClientSecret
						}
					case fieldMcpToken:
						if cfg.Token != nil {
							currentVal = *cfg.Token
						}
					case fieldMcpCwd:
						currentVal = cfg.Cwd
					}
					m.input.SetValue(currentVal)
				}
			case "u":
				f := mcpFields[m.fieldCursor]
				cfg := m.config.McpServers[m.activeServer]
				switch f.key {
				case fieldMcpCommand:
					cfg.Command = ""
				case fieldMcpArgs:
					cfg.Args = nil
				case fieldMcpUrl:
					cfg.Url = ""
				case fieldMcpClientID:
					cfg.ClientID = nil
				case fieldMcpClientSecret:
					cfg.ClientSecret = nil
				case fieldMcpToken:
					cfg.Token = nil
				case fieldMcpCwd:
					cfg.Cwd = ""
				case fieldMcpDisabled:
					cfg.Disabled = false
				}
				m.config.McpServers[m.activeServer] = cfg
			case "q", "esc":
				m.state = stateMcpListServers
			}

		case stateMcpEditFieldValue:
			switch msg.String() {
			case "enter":
				newVal := m.input.Value()
				cfg := m.config.McpServers[m.activeServer]
				switch m.editingField {
				case fieldMcpName:
					newVal = strings.TrimSpace(newVal)
					if newVal != "" && newVal != m.activeServer {
						delete(m.config.McpServers, m.activeServer)
						m.config.McpServers[newVal] = cfg
						m.activeServer = newVal
						m.updateServerNames()
						for i, s := range m.servers {
							if s == newVal {
								m.listCursor = i
								break
							}
						}
					}
				case fieldMcpCommand:
					cfg.Command = newVal
					m.config.McpServers[m.activeServer] = cfg
				case fieldMcpArgs:
					if strings.TrimSpace(newVal) == "" {
						cfg.Args = nil
					} else {
						cfg.Args = strings.Fields(newVal)
					}
					m.config.McpServers[m.activeServer] = cfg
				case fieldMcpUrl:
					cfg.Url = newVal
					m.config.McpServers[m.activeServer] = cfg
				case fieldMcpClientID:
					if newVal == "" {
						cfg.ClientID = nil
					} else {
						cfg.ClientID = &newVal
					}
					m.config.McpServers[m.activeServer] = cfg
				case fieldMcpClientSecret:
					if newVal == "" {
						cfg.ClientSecret = nil
					} else {
						cfg.ClientSecret = &newVal
					}
					m.config.McpServers[m.activeServer] = cfg
				case fieldMcpToken:
					if newVal == "" {
						cfg.Token = nil
					} else {
						cfg.Token = &newVal
					}
					m.config.McpServers[m.activeServer] = cfg
				case fieldMcpCwd:
					cfg.Cwd = newVal
					m.config.McpServers[m.activeServer] = cfg
				}
				m.state = stateMcpEditServerFields
			case "esc":
				m.state = stateMcpEditServerFields
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m interactiveMcpModel) View() string {
	var s strings.Builder

	s.WriteString("\n" + StyleHeader.Render("⚡ FRAGS MCP SERVER EDITOR ⚡") + "\n\n")

	switch m.state {
	case stateMcpChooseFile:
		s.WriteString(RenderFileSelection("Select which tools configuration file to edit:", m.fileCursor, m.fileOptions))

	case stateMcpListServers:
		s.WriteString(fmt.Sprintf("  File: %s\n\n", StyleValActive.Render(m.filePath)))
		s.WriteString("  Configured MCP Servers:\n\n")

		if len(m.servers) == 0 {
			s.WriteString(StyleWillCreate.Render("  No MCP servers configured yet. Press 'a' to add one.\n\n"))
		} else {
			for i, sName := range m.servers {
				cursor := "  "
				nameStyle := StyleLabelDefault
				infoStyle := StyleMuted

				if i == m.listCursor {
					cursor = StyleCursor.Render("> ")
					nameStyle = StyleLabelActive
					infoStyle = StyleValDefault
				}

				srv := m.config.McpServers[sName]
				var status string
				if srv.Disabled {
					status = StyleWillCreate.Render("(disabled)")
				} else {
					status = StyleExists.Render("(enabled)")
				}

				var details string
				if srv.Url != "" {
					details = fmt.Sprintf("Url: %s", srv.Url)
				} else if srv.Command != "" {
					argsStr := strings.Join(srv.Args, " ")
					details = fmt.Sprintf("Command: %s %s", srv.Command, argsStr)
				} else {
					details = "No URL or Command configured"
				}

				s.WriteString(fmt.Sprintf("%s%-25s %s\n", cursor, nameStyle.Render(sName), status))
				s.WriteString(fmt.Sprintf("    %s\n\n", infoStyle.Render(details)))
			}
		}

		s.WriteString(StyleHelp.Render("  [a] Add Server  |  [d] Delete  |  [s] Save & Exit  |  [esc/q] Exit without saving") + "\n")

	case stateMcpEditServerFields:
		s.WriteString(fmt.Sprintf("  Server: %s\n\n", StyleLabelActive.Render(m.activeServer)))
		s.WriteString("  Edit fields:\n\n")

		cfg := m.config.McpServers[m.activeServer]

		for i, f := range mcpFields {
			cursor := "  "
			fieldStyle := StyleLabelDefault
			valStyle := StyleValDefault

			if i == m.fieldCursor {
				cursor = StyleCursor.Render("> ")
				fieldStyle = StyleLabelActive
				valStyle = StyleValActive
			}

			var val string
			switch f.key {
			case fieldMcpName:
				val = m.activeServer
			case fieldMcpCommand:
				val = cfg.Command
			case fieldMcpArgs:
				val = strings.Join(cfg.Args, " ")
			case fieldMcpUrl:
				val = cfg.Url
			case fieldMcpDisabled:
				if cfg.Disabled {
					val = "true"
				} else {
					val = "false"
				}
			case fieldMcpClientID:
				if cfg.ClientID != nil {
					val = *cfg.ClientID
				}
			case fieldMcpClientSecret:
				if cfg.ClientSecret != nil {
					val = *cfg.ClientSecret
				}
			case fieldMcpToken:
				if cfg.Token != nil {
					val = *cfg.Token
				}
			case fieldMcpCwd:
				val = cfg.Cwd
			}

			if val == "" {
				if f.key == fieldMcpDisabled {
					val = "false"
				} else {
					val = StyleMuted.Render("<not set>")
				}
			}

			s.WriteString(fmt.Sprintf("%s%-18s = %s\n", cursor, fieldStyle.Render(f.label), valStyle.Render(val)))
		}

		activeField := mcpFields[m.fieldCursor]

		s.WriteString("\n" + StyleDescBox.Render(
			fmt.Sprintf("%s\n%s",
				StyleValActive.Render(activeField.label),
				StyleLabelDefault.Render(activeField.description),
			),
		) + "\n\n")

		s.WriteString(StyleHelp.Render("  (Use Up/Down arrows, Enter to edit/toggle, u to unset, esc/q to go back)") + "\n")

	case stateMcpEditFieldValue:
		activeField := mcpFields[m.fieldCursor]
		s.WriteString(RenderTextInputField(
			fmt.Sprintf("Server: %s", StyleExists.Render(m.activeServer)),
			activeField.label,
			m.input.View(),
		))
	}

	return s.String()
}

func runInteractiveMcp(preselectedPath string) error {
	m := newInteractiveMcpModel(preselectedPath)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	mResult := finalModel.(interactiveMcpModel)
	if mResult.err != nil {
		return mResult.err
	}
	if mResult.saved {
		fmt.Printf("\nSuccessfully saved configuration to: %s\n", mResult.filePath)
	}
	return nil
}
