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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fileOption struct {
	label string
	path  string
}

// Color definitions for UI consistency
var (
	ColorWhite    = lipgloss.Color("#FAFAFA")
	ColorViolet   = lipgloss.Color("#7D56F4")
	ColorHotPink  = lipgloss.Color("#FF007F")
	ColorGreen    = lipgloss.Color("#00FF87")
	ColorBlue     = lipgloss.Color("#00AFFF")
	ColorMuted    = lipgloss.Color("#585858")
	ColorGray     = lipgloss.Color("#BCBCBC")
	ColorGrayDark = lipgloss.Color("#8A8A8A")
	ColorHelp     = lipgloss.Color("#767676")
	ColorOrange   = lipgloss.Color("#FFAF00")

	// Component specific colors
	ColorCompRunner      = lipgloss.Color("#5F87FF")
	ColorCompWorker      = lipgloss.Color("#00AFFF")
	ColorCompFunction    = lipgloss.Color("#00FF87")
	ColorCompTransformer = lipgloss.Color("#FFAF00")
	ColorCompSession     = lipgloss.Color("#AF5FFF")
	ColorCompPrePrompt   = lipgloss.Color("#0087AF")
	ColorCompPrompt      = lipgloss.Color("#00AFD7")
	ColorCompAi          = lipgloss.Color("#FF00FF")
	ColorCompApp         = lipgloss.Color("#8A8A8A")
	ColorCompMcp         = lipgloss.Color("#D787FF")
)

// Shared reusable Lipgloss Styles
var (
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorViolet).
			Padding(0, 2).
			MarginBottom(1)

	StyleCursor = lipgloss.NewStyle().Foreground(ColorHotPink)

	StyleLabelDefault = lipgloss.NewStyle().Foreground(ColorGray)
	StyleLabelActive  = lipgloss.NewStyle().Bold(true).Foreground(ColorHotPink)

	StyleValDefault = lipgloss.NewStyle().Foreground(ColorBlue)
	StyleValActive  = lipgloss.NewStyle().Bold(true).Foreground(ColorGreen)

	StyleMuted = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleHelp  = lipgloss.NewStyle().Foreground(ColorHelp)

	StyleExists     = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleWillCreate = lipgloss.NewStyle().Foreground(ColorGrayDark)

	StyleDescBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorViolet).
			Padding(0, 1).
			Width(65).
			MarginTop(1)

	StyleDescTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorGreen)
	StyleDescText  = lipgloss.NewStyle().Foreground(ColorGray)

	StyleOptCustom = lipgloss.NewStyle().Bold(true).Foreground(ColorOrange)

	// Runner/Activity Styles
	StyleRunnerSuccess = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	StyleRunnerError   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)
	StyleDivider       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#BCBCBC", Dark: "#3A3A3A"})

	StyleEventTime         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#767676", Dark: "#585858"})
	StyleEventPrefixStart  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	StyleEventPrefixEnd    = lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF"))
	StyleEventPrefixError  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	StyleEventPrefixResult = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66"))
	StyleEventDetails      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#767676", Dark: "#585858"})
	StyleEventMsgError     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))
	StyleEventMsgDefault   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1C1C1C", Dark: "#D0D0D0"})

	StyleActiveSessionsHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#0087AF", Dark: "#00D7FF"})
	StyleSessionIcon          = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleSessionID            = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#0087AF", Dark: "#00D7FF"})
	StyleSessionComp          = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D7005F", Dark: "#FF007F"})
	StyleSessionText          = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1C1C1C", Dark: "#E5E5E5"})

	StyleActivityLogsHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#875FDF", Dark: "#AF5FFF"})

	StyleFooterDefault = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#767676", Dark: "#585858"})
	StyleFooterError   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3333"))
	StyleFooterSuccess = lipgloss.NewStyle().Bold(true).Foreground(ColorGreen)
)

// Reusable Procedural UI Components

// UpdateFileSelection handles standard file chooser arrow navigation and confirm/quit keyboard input.
func UpdateFileSelection(key string, cursor *int, options []fileOption) (path string, completed bool, quit bool) {
	switch key {
	case "up", "k":
		if *cursor > 0 {
			*cursor--
		}
	case "down", "j":
		if *cursor < len(options)-1 {
			*cursor++
		}
	case "enter":
		return options[*cursor].path, true, false
	case "q", "esc":
		return "", false, true
	}
	return "", false, false
}

// RenderFileSelection renders a standard interactive list of files to select.
func RenderFileSelection(title string, cursor int, options []fileOption) string {
	var s strings.Builder
	s.WriteString("  " + title + "\n\n")
	for i, opt := range options {
		indicator := "  "
		labelStyle := StyleLabelDefault
		if i == cursor {
			indicator = StyleCursor.Render("> ")
			labelStyle = StyleLabelActive
		}

		existsStr := StyleExists.Render("(exists)")
		if _, err := os.Stat(opt.path); os.IsNotExist(err) {
			existsStr = StyleWillCreate.Render("(will create)")
		}

		s.WriteString(fmt.Sprintf("%s%s %s\n", indicator, labelStyle.Render(opt.label), existsStr))
		s.WriteString(fmt.Sprintf("    %s\n\n", StyleMuted.Render(opt.path)))
	}
	s.WriteString(StyleHelp.Render("  (Use Up/Down arrows, Enter to select, q/esc to quit)") + "\n")
	return s.String()
}

// RenderTextInputField renders a uniform interface for editing a single value via textinput.
func RenderTextInputField(contextHeader, fieldLabel string, inputView string) string {
	var s strings.Builder
	if contextHeader != "" {
		s.WriteString(fmt.Sprintf("  %s\n", contextHeader))
	}
	s.WriteString(fmt.Sprintf("  Editing: %s\n\n", StyleLabelActive.Render(fieldLabel)))
	s.WriteString("  Enter new value:\n\n")
	s.WriteString("  " + inputView + "\n\n")
	s.WriteString(StyleHelp.Render("  (Press Enter to save, Esc to cancel)") + "\n")
	return s.String()
}

type fileChooserModel struct {
	title    string
	options  []fileOption
	cursor   int
	selected string
	quit     bool
}

func (m fileChooserModel) Init() tea.Cmd {
	return nil
}

func (m fileChooserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		path, completed, quit := UpdateFileSelection(msg.String(), &m.cursor, m.options)
		if quit {
			m.quit = true
			return m, tea.Quit
		}
		if completed {
			m.selected = path
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m fileChooserModel) View() string {
	var s strings.Builder
	s.WriteString("\n" + StyleHeader.Render("⚡ SELECT FILE ⚡") + "\n\n")
	s.WriteString(RenderFileSelection(m.title, m.cursor, m.options))
	return s.String()
}

func ChooseFile(title string, options []fileOption) (string, error) {
	m := fileChooserModel{
		title:   title,
		options: options,
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	res, err := p.Run()
	if err != nil {
		return "", err
	}
	finalModel := res.(fileChooserModel)
	if finalModel.quit {
		return "", fmt.Errorf("cancelled")
	}
	return finalModel.selected, nil
}
