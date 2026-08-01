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
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theirish81/frags"
)

type collEditState int

const (
	stateCollChooseFile collEditState = iota
	stateCollListCollections
	stateCollEditCollectionFields
	stateCollEditFieldValue
	stateCollListParams
	stateCollAddParamKey
	stateCollRenameParamKey
	stateCollEditParamValue
)

type collFieldKey string

const (
	fieldCollName     collFieldKey = "Name"
	fieldCollToolType collFieldKey = "ToolType"
	fieldCollDisabled collFieldKey = "Disabled"
	fieldCollParams   collFieldKey = "Params"
)

type collField struct {
	key         collFieldKey
	label       string
	description string
}

var collFields = []collField{
	{fieldCollName, "Collection Name", "Unique identifier for this tools collection"},
	{fieldCollToolType, "Tool Type", "The type of the tool (e.g. browser, file, web)"},
	{fieldCollDisabled, "Disabled", "Toggle whether this tools collection is disabled"},
	{fieldCollParams, "Params", "Navigate to configure individual parameters key-value list"},
}

type interactiveCollectionsModel struct {
	state       collEditState
	fileOptions []fileOption
	fileCursor  int

	filePath string
	config   ExtendedToolsConfig
	colls    []string

	listCursor  int
	fieldCursor int
	input       textinput.Model

	activeColl   string
	editingField collFieldKey

	paramKeys   []string
	paramCursor int
	activeParam string

	saved bool
	err   error
}

func newInteractiveCollectionsModel(preselectedPath string) interactiveCollectionsModel {
	var fileOpts []fileOption
	fileOpts = append(fileOpts, fileOption{
		label: "Current Directory (tools.json)",
		path:  "tools.json",
	})
	if home, err := os.UserHomeDir(); err == nil {
		fileOpts = append(fileOpts, fileOption{
			label: "User Global Config (~/.config/frags/tools.json)",
			path:  filepath.Join(home, ".config", "frags", "tools.json"),
		})
	}

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	m := interactiveCollectionsModel{
		state:       stateCollChooseFile,
		fileOptions: fileOpts,
		input:       ti,
	}

	if preselectedPath != "" {
		m.filePath = preselectedPath
		m.state = stateCollListCollections
		m.loadConfig()
	}

	return m
}

func (m *interactiveCollectionsModel) loadConfig() {
	data, err := os.ReadFile(m.filePath)
	if os.IsNotExist(err) {
		m.config = ExtendedToolsConfig{}
		if m.config.Collections == nil {
			m.config.Collections = make(frags.ToolsCollectionConfigs)
		}
	} else if err != nil {
		m.err = err
		return
	} else {
		m.config, err = parseToolsConfig(data)
		if err != nil {
			m.err = err
			return
		}
	}
	if m.config.Collections == nil {
		m.config.Collections = make(frags.ToolsCollectionConfigs)
	}
	m.updateCollectionNames()
}

func (m *interactiveCollectionsModel) updateCollectionNames() {
	var keys []string
	for k := range m.config.Collections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	m.colls = keys
}

func (m *interactiveCollectionsModel) updateParamKeys() {
	cfg := m.config.Collections[m.activeColl]
	var keys []string
	for k := range cfg.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	m.paramKeys = keys
}

func (m interactiveCollectionsModel) Init() tea.Cmd {
	return nil
}

func (m interactiveCollectionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state != stateCollEditFieldValue && m.state != stateCollAddParamKey && m.state != stateCollRenameParamKey && m.state != stateCollEditParamValue {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
		}

		switch m.state {
		case stateCollChooseFile:
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
				m.loadConfig()
				if m.err != nil {
					return m, tea.Quit
				}
				m.state = stateCollListCollections
			case "q", "esc":
				return m, tea.Quit
			}

		case stateCollListCollections:
			switch msg.String() {
			case "up", "k":
				if m.listCursor > 0 {
					m.listCursor--
				}
			case "down", "j":
				if m.listCursor < len(m.colls)-1 {
					m.listCursor++
				}
			case "enter":
				if len(m.colls) > 0 {
					m.activeColl = m.colls[m.listCursor]
					m.state = stateCollEditCollectionFields
					m.fieldCursor = 0
				}
			case "a":
				newName := "new_collection"
				index := 1
				for {
					candidate := fmt.Sprintf("%s_%d", newName, index)
					if _, exists := m.config.Collections[candidate]; !exists {
						newName = candidate
						break
					}
					index++
				}
				m.config.Collections[newName] = frags.CollectionConfig{
					ToolType: "web",
					Params:   make(map[string]string),
				}
				m.updateCollectionNames()
				for idx, name := range m.colls {
					if name == newName {
						m.listCursor = idx
						break
					}
				}
			case "d":
				if len(m.colls) > 0 {
					target := m.colls[m.listCursor]
					delete(m.config.Collections, target)
					m.updateCollectionNames()
					if m.listCursor >= len(m.colls) && m.listCursor > 0 {
						m.listCursor = len(m.colls) - 1
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

		case stateCollEditCollectionFields:
			switch msg.String() {
			case "up", "k":
				if m.fieldCursor > 0 {
					m.fieldCursor--
				}
			case "down", "j":
				if m.fieldCursor < len(collFields)-1 {
					m.fieldCursor++
				}
			case "enter":
				f := collFields[m.fieldCursor]
				cfg := m.config.Collections[m.activeColl]
				if f.key == fieldCollDisabled {
					cfg.Disabled = !cfg.Disabled
					m.config.Collections[m.activeColl] = cfg
				} else if f.key == fieldCollParams {
					m.updateParamKeys()
					m.paramCursor = 0
					m.state = stateCollListParams
				} else {
					m.editingField = f.key
					m.state = stateCollEditFieldValue
					var currentVal string
					switch f.key {
					case fieldCollName:
						currentVal = m.activeColl
					case fieldCollToolType:
						currentVal = cfg.ToolType
					}
					m.input.SetValue(currentVal)
				}
			case "u":
				f := collFields[m.fieldCursor]
				cfg := m.config.Collections[m.activeColl]
				switch f.key {
				case fieldCollToolType:
					cfg.ToolType = ""
				case fieldCollParams:
					cfg.Params = make(map[string]string)
				case fieldCollDisabled:
					cfg.Disabled = false
				}
				m.config.Collections[m.activeColl] = cfg
			case "q", "esc":
				m.state = stateCollListCollections
			}

		case stateCollEditFieldValue:
			switch msg.String() {
			case "enter":
				newVal := m.input.Value()
				cfg := m.config.Collections[m.activeColl]
				switch m.editingField {
				case fieldCollName:
					newVal = strings.TrimSpace(newVal)
					if newVal != "" && newVal != m.activeColl {
						delete(m.config.Collections, m.activeColl)
						m.config.Collections[newVal] = cfg
						m.activeColl = newVal
						m.updateCollectionNames()
						for idx, name := range m.colls {
							if name == newVal {
								m.listCursor = idx
								break
							}
						}
					}
				case fieldCollToolType:
					cfg.ToolType = newVal
					m.config.Collections[m.activeColl] = cfg
				}
				m.state = stateCollEditCollectionFields
			case "esc":
				m.state = stateCollEditCollectionFields
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

		case stateCollListParams:
			switch msg.String() {
			case "up", "k":
				if m.paramCursor > 0 {
					m.paramCursor--
				}
			case "down", "j":
				if m.paramCursor < len(m.paramKeys)-1 {
					m.paramCursor++
				}
			case "enter":
				if len(m.paramKeys) > 0 {
					m.activeParam = m.paramKeys[m.paramCursor]
					cfg := m.config.Collections[m.activeColl]
					m.input.SetValue(cfg.Params[m.activeParam])
					m.state = stateCollEditParamValue
				}
			case "a":
				m.input.SetValue("")
				m.state = stateCollAddParamKey
			case "d":
				if len(m.paramKeys) > 0 {
					target := m.paramKeys[m.paramCursor]
					cfg := m.config.Collections[m.activeColl]
					delete(cfg.Params, target)
					m.config.Collections[m.activeColl] = cfg
					m.updateParamKeys()
					if m.paramCursor >= len(m.paramKeys) && m.paramCursor > 0 {
						m.paramCursor = len(m.paramKeys) - 1
					}
				}
			case "r":
				if len(m.paramKeys) > 0 {
					m.activeParam = m.paramKeys[m.paramCursor]
					m.input.SetValue(m.activeParam)
					m.state = stateCollRenameParamKey
				}
			case "q", "esc":
				m.state = stateCollEditCollectionFields
			}

		case stateCollAddParamKey:
			switch msg.String() {
			case "enter":
				newKey := strings.TrimSpace(m.input.Value())
				if newKey != "" {
					cfg := m.config.Collections[m.activeColl]
					if cfg.Params == nil {
						cfg.Params = make(map[string]string)
					}
					if _, exists := cfg.Params[newKey]; !exists {
						cfg.Params[newKey] = ""
						m.config.Collections[m.activeColl] = cfg
						m.updateParamKeys()
						for idx, k := range m.paramKeys {
							if k == newKey {
								m.paramCursor = idx
								break
							}
						}
					}
				}
				m.state = stateCollListParams
			case "esc":
				m.state = stateCollListParams
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

		case stateCollRenameParamKey:
			switch msg.String() {
			case "enter":
				newKey := strings.TrimSpace(m.input.Value())
				if newKey != "" && newKey != m.activeParam {
					cfg := m.config.Collections[m.activeColl]
					if cfg.Params != nil {
						val := cfg.Params[m.activeParam]
						delete(cfg.Params, m.activeParam)
						cfg.Params[newKey] = val
						m.config.Collections[m.activeColl] = cfg
						m.updateParamKeys()
						for idx, k := range m.paramKeys {
							if k == newKey {
								m.paramCursor = idx
								break
							}
						}
					}
				}
				m.state = stateCollListParams
			case "esc":
				m.state = stateCollListParams
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

		case stateCollEditParamValue:
			switch msg.String() {
			case "enter":
				newVal := m.input.Value()
				cfg := m.config.Collections[m.activeColl]
				if cfg.Params != nil {
					cfg.Params[m.activeParam] = newVal
					m.config.Collections[m.activeColl] = cfg
				}
				m.state = stateCollListParams
			case "esc":
				m.state = stateCollListParams
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m interactiveCollectionsModel) View() string {
	var s strings.Builder

	s.WriteString("\n" + StyleHeader.Render("⚡ FRAGS TOOLS COLLECTIONS EDITOR ⚡") + "\n\n")

	switch m.state {
	case stateCollChooseFile:
		s.WriteString("  Select which tools configuration file to edit:\n\n")
		for i, opt := range m.fileOptions {
			cursor := "  "
			labelStyle := StyleLabelDefault
			if i == m.fileCursor {
				cursor = StyleCursor.Render("> ")
				labelStyle = StyleLabelActive
			}

			existsStr := StyleExists.Render("(exists)")
			if _, err := os.Stat(opt.path); os.IsNotExist(err) {
				existsStr = StyleWillCreate.Render("(will create)")
			}

			s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, labelStyle.Render(opt.label), existsStr))
			s.WriteString(fmt.Sprintf("    %s\n\n", StyleMuted.Render(opt.path)))
		}
		s.WriteString(StyleHelp.Render("  (Use Up/Down arrows, Enter to select, q/esc to quit)") + "\n")

	case stateCollListCollections:
		s.WriteString(fmt.Sprintf("  File: %s\n\n", StyleValActive.Render(m.filePath)))
		s.WriteString("  Configured Tools Collections:\n\n")

		if len(m.colls) == 0 {
			s.WriteString(StyleWillCreate.Render("  No collections configured yet. Press 'a' to add one.\n\n"))
		} else {
			for i, cName := range m.colls {
				cursor := "  "
				nameStyle := StyleLabelDefault
				infoStyle := StyleMuted

				if i == m.listCursor {
					cursor = StyleCursor.Render("> ")
					nameStyle = StyleLabelActive
					infoStyle = StyleValDefault
				}

				cfg := m.config.Collections[cName]
				var status string
				if cfg.Disabled {
					status = StyleWillCreate.Render("(disabled)")
				} else {
					status = StyleExists.Render("(enabled)")
				}

				var params []string
				var pKeys []string
				for pk := range cfg.Params {
					pKeys = append(pKeys, pk)
				}
				sort.Strings(pKeys)
				for _, pk := range pKeys {
					params = append(params, fmt.Sprintf("%s=%s", pk, cfg.Params[pk]))
				}
				paramsStr := strings.Join(params, ", ")
				if paramsStr == "" {
					paramsStr = "<none>"
				}

				details := fmt.Sprintf("Type: %s | Params: %s", cfg.ToolType, paramsStr)

				s.WriteString(fmt.Sprintf("%s%-25s %s\n", cursor, nameStyle.Render(cName), status))
				s.WriteString(fmt.Sprintf("    %s\n\n", infoStyle.Render(details)))
			}
		}

		s.WriteString(StyleHelp.Render("  [a] Add Collection  |  [d] Delete  |  [s] Save & Exit  |  [esc/q] Exit without saving") + "\n")

	case stateCollEditCollectionFields:
		s.WriteString(fmt.Sprintf("  Collection: %s\n\n", StyleLabelActive.Render(m.activeColl)))
		s.WriteString("  Edit fields:\n\n")

		cfg := m.config.Collections[m.activeColl]

		for i, f := range collFields {
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
			case fieldCollName:
				val = m.activeColl
			case fieldCollToolType:
				val = cfg.ToolType
			case fieldCollDisabled:
				if cfg.Disabled {
					val = "true"
				} else {
					val = "false"
				}
			case fieldCollParams:
				var params []string
				var pKeys []string
				for pk := range cfg.Params {
					pKeys = append(pKeys, pk)
				}
				sort.Strings(pKeys)
				for _, pk := range pKeys {
					params = append(params, fmt.Sprintf("%s=%s", pk, cfg.Params[pk]))
				}
				val = strings.Join(params, " ")
			}

			if val == "" {
				if f.key == fieldCollDisabled {
					val = "false"
				} else {
					val = StyleMuted.Render("<not set>")
				}
			}

			s.WriteString(fmt.Sprintf("%s%-18s = %s\n", cursor, fieldStyle.Render(f.label), valStyle.Render(val)))
		}

		activeField := collFields[m.fieldCursor]

		s.WriteString("\n" + StyleDescBox.Render(
			fmt.Sprintf("%s\n%s",
				StyleValActive.Render(activeField.label),
				StyleLabelDefault.Render(activeField.description),
			),
		) + "\n\n")

		s.WriteString(StyleHelp.Render("  (Use Up/Down arrows, Enter to edit/toggle, u to unset, esc/q to go back)") + "\n")

	case stateCollEditFieldValue:
		activeField := collFields[m.fieldCursor]
		s.WriteString(fmt.Sprintf("  Collection: %s\n", StyleExists.Render(m.activeColl)))
		s.WriteString(fmt.Sprintf("  Editing: %s\n\n", StyleLabelActive.Render(activeField.label)))

		s.WriteString("  Enter new value:\n\n")
		s.WriteString("  " + m.input.View() + "\n\n")

		s.WriteString(StyleHelp.Render("  (Press Enter to save, Esc to cancel)") + "\n")

	case stateCollListParams:
		s.WriteString(fmt.Sprintf("  Collection: %s\n\n", StyleExists.Render(m.activeColl)))
		s.WriteString("  Collection Parameters:\n\n")

		cfg := m.config.Collections[m.activeColl]
		if len(m.paramKeys) == 0 {
			s.WriteString(StyleWillCreate.Render("  No parameters configured yet. Press 'a' to add one.\n\n"))
		} else {
			for i, pk := range m.paramKeys {
				cursor := "  "
				keyStyle := StyleLabelDefault
				valStyle := StyleValDefault

				if i == m.paramCursor {
					cursor = StyleCursor.Render("> ")
					keyStyle = StyleLabelActive
					valStyle = StyleValActive
				}

				val := cfg.Params[pk]
				if val == "" {
					val = StyleMuted.Render("<not set>")
				} else {
					val = fmt.Sprintf(`"%s"`, val)
				}

				s.WriteString(fmt.Sprintf("%s%-20s = %s\n", cursor, keyStyle.Render(pk), valStyle.Render(val)))
			}
			s.WriteString("\n")
		}

		s.WriteString(StyleHelp.Render("  [a] Add Param  |  [d] Delete  |  [r] Rename Key  |  [enter] Edit Value  |  [esc/q] Go Back") + "\n")

	case stateCollAddParamKey:
		s.WriteString(fmt.Sprintf("  Collection: %s\n\n", StyleExists.Render(m.activeColl)))
		s.WriteString("  Add Parameter Key:\n\n")
		s.WriteString("  " + m.input.View() + "\n\n")
		s.WriteString(StyleHelp.Render("  (Press Enter to create, Esc to cancel)") + "\n")

	case stateCollRenameParamKey:
		s.WriteString(fmt.Sprintf("  Collection: %s\n", StyleExists.Render(m.activeColl)))
		s.WriteString(fmt.Sprintf("  Renaming Parameter Key: %s\n\n", StyleLabelActive.Render(m.activeParam)))
		s.WriteString("  Enter new key name:\n\n")
		s.WriteString("  " + m.input.View() + "\n\n")
		s.WriteString(StyleHelp.Render("  (Press Enter to rename, Esc to cancel)") + "\n")

	case stateCollEditParamValue:
		s.WriteString(fmt.Sprintf("  Collection: %s\n", StyleExists.Render(m.activeColl)))
		s.WriteString(fmt.Sprintf("  Editing Parameter: %s\n\n", StyleLabelActive.Render(m.activeParam)))
		s.WriteString("  Enter value:\n\n")
		s.WriteString("  " + m.input.View() + "\n\n")
		s.WriteString(StyleHelp.Render("  (Press Enter to save, Esc to cancel)") + "\n")
	}

	return s.String()
}

func runInteractiveCollections(preselectedPath string) error {
	m := newInteractiveCollectionsModel(preselectedPath)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	mResult := finalModel.(interactiveCollectionsModel)
	if mResult.err != nil {
		return mResult.err
	}
	if mResult.saved {
		fmt.Printf("\nSuccessfully saved tools configuration to: %s\n", mResult.filePath)
	}
	return nil
}
