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
