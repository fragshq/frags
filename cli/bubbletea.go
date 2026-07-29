/*
 * Copyright (C) 2026 Simone Pezzano
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
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/util"
)

type runResult struct {
	result util.ProgMap
	err    error
}

type sessionState struct {
	lastActivity string
	component    string
	startTime    time.Time
}

type model struct {
	spinner        spinner.Model
	eventChan      chan log.Event
	resChan        chan runResult
	events         []log.Event
	activeSessions map[string]sessionState
	done           bool
	result         util.ProgMap
	err            error
	width          int
	height         int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		listenEvents(m.eventChan),
		listenResult(m.resChan),
	)
}

func listenEvents(ch chan log.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ev
	}
}

func listenResult(ch chan runResult) tea.Cmd {
	return func() tea.Msg {
		res, ok := <-ch
		if !ok {
			return nil
		}
		return res
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			m.err = fmt.Errorf("interrupted by user")
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case log.Event:
		m.events = append(m.events, msg)
		if len(m.events) > 8 {
			m.events = m.events[1:]
		}

		if msg.Session != nil {
			sessID := *msg.Session
			if msg.Component == log.SessionComponent && msg.Type == log.EndEventType {
				delete(m.activeSessions, sessID)
			} else {
				state := m.activeSessions[sessID]
				if msg.Message != "" {
					state.lastActivity = msg.Message
				}
				state.component = string(msg.Component)
				if state.startTime.IsZero() {
					state.startTime = msg.Time
				}
				m.activeSessions[sessID] = state
			}
		}

		return m, listenEvents(m.eventChan)

	case runResult:
		m.done = true
		m.result = msg.result
		m.err = msg.err
		return m, tea.Quit
	}

	return m, nil
}

func styleComponent(comp log.EventComponent) string {
	s := lipgloss.NewStyle().Bold(true)
	switch comp {
	case log.RunnerComponent:
		return s.Foreground(lipgloss.Color("#5F87FF")).Render("runner")
	case log.WorkerComponent:
		return s.Foreground(lipgloss.Color("#00AFFF")).Render("worker")
	case log.FunctionComponent:
		return s.Foreground(lipgloss.Color("#00FF87")).Render("function")
	case log.TransformerComponent:
		return s.Foreground(lipgloss.Color("#FFAF00")).Render("transformer")
	case log.SessionComponent:
		return s.Foreground(lipgloss.Color("#AF5FFF")).Render("session")
	case log.PrePromptComponent:
		return s.Foreground(lipgloss.Color("#0087AF")).Render("pre-prompt")
	case log.PromptComponent:
		return s.Foreground(lipgloss.Color("#00AFD7")).Render("prompt")
	case log.AiComponent:
		return s.Foreground(lipgloss.Color("#FF00FF")).Render("ai")
	case log.AppComponent:
		return s.Foreground(lipgloss.Color("#8A8A8A")).Render("app")
	case log.McpComponent:
		return s.Foreground(lipgloss.Color("#D787FF")).Render("mcp")
	default:
		return s.Foreground(lipgloss.Color("#BCBCBC")).Render(string(comp))
	}
}

func formatEventLine(ev log.Event) string {
	timeStr := ev.Time.Format("15:04:05")
	timeStyled := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#767676", Dark: "#585858"}).Render(timeStr)

	compStyled := styleComponent(ev.Component)

	var prefix string
	switch ev.Type {
	case log.StartEventType:
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("▶")
	case log.EndEventType:
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF")).Render("◀")
	case log.ErrorEventType:
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("✘")
	case log.ResultEventType:
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66")).Render("✔")
	default:
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")).Render("●")
	}

	msg := ev.Message
	if msg == "" && ev.Err != nil {
		msg = ev.Err.Error()
	}

	var details []string
	if ev.Session != nil && *ev.Session != "" {
		details = append(details, fmt.Sprintf("session=%s", *ev.Session))
	}
	if ev.Function != nil && *ev.Function != "" {
		details = append(details, fmt.Sprintf("fn=%s", *ev.Function))
	}
	if ev.Resource != nil && *ev.Resource != "" {
		details = append(details, fmt.Sprintf("res=%s", *ev.Resource))
	}
	if ev.Iteration != nil {
		details = append(details, fmt.Sprintf("iter=%d", *ev.Iteration))
	}
	if ev.Phase != nil {
		details = append(details, fmt.Sprintf("phase=%d", *ev.Phase))
	}
	if ev.Transformer != nil && *ev.Transformer != "" {
		details = append(details, fmt.Sprintf("transformer=%s", *ev.Transformer))
	}
	if ev.Engine != nil && *ev.Engine != "" {
		details = append(details, fmt.Sprintf("engine=%s", *ev.Engine))
	}
	if ev.Content != nil && *ev.Content != nil {
		contentStr := fmt.Sprintf("%v", *ev.Content)
		if contentStr != "" {
			details = append(details, fmt.Sprintf("content=%s", contentStr))
		}
	}
	if ev.Err != nil && ev.Message != "" {
		details = append(details, fmt.Sprintf("error=%s", ev.Err.Error()))
	}
	for k, val := range ev.Args {
		details = append(details, fmt.Sprintf("%s=%v", k, val))
	}

	detailsStr := ""
	if len(details) > 0 {
		detailsStr = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#767676", Dark: "#585858"}).Render(" (" + strings.Join(details, ", ") + ")")
	}

	var msgStyle lipgloss.Style
	if ev.Type == log.ErrorEventType {
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))
	} else {
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1C1C1C", Dark: "#D0D0D0"})
	}

	return fmt.Sprintf("%s %s %s: %s%s", timeStyled, prefix, compStyled, msgStyle.Render(msg), detailsStr)
}

func (m model) View() string {
	var s strings.Builder

	header := " ⚡ FRAGS RUNNER "
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	s.WriteString(headerStyle.Render(header) + " ")

	if !m.done {
		s.WriteString(m.spinner.View() + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("Executing plan..."))
	} else {
		if m.err != nil {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true).Render("✘ Failed"))
		} else {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66")).Bold(true).Render("✔ Succeeded"))
		}
	}
	s.WriteString("\n\n")

	dividerWidth := 60
	if m.width > 0 && m.width < 60 {
		dividerWidth = m.width - 2
	} else if m.width >= 60 {
		dividerWidth = m.width - 4
	}
	divider := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#BCBCBC", Dark: "#3A3A3A"}).Render(strings.Repeat("─", dividerWidth))

	if len(m.activeSessions) > 0 {
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#0087AF", Dark: "#00D7FF"}).Render("Active Sessions:") + "\n")
		for id, state := range m.activeSessions {
			sessIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87")).Render("●")
			line := fmt.Sprintf("  %s %s [%s]: %s",
				sessIcon,
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#0087AF", Dark: "#00D7FF"}).Render(id),
				lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D7005F", Dark: "#FF007F"}).Render(state.component),
				lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1C1C1C", Dark: "#E5E5E5"}).Render(state.lastActivity))
			if m.width > 0 {
				line = lipgloss.NewStyle().Width(m.width - 2).Render(line)
			}
			s.WriteString(line + "\n")
		}
		s.WriteString(divider + "\n")
	}

	if len(m.events) > 0 {
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#875FDF", Dark: "#AF5FFF"}).Render("Activity Logs:") + "\n")
		for _, ev := range m.events {
			line := formatEventLine(ev)
			if m.width > 0 {
				line = lipgloss.NewStyle().Width(m.width - 4).Render(line)
			}
			s.WriteString("  " + line + "\n")
		}
		s.WriteString(divider + "\n")
	}

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#767676", Dark: "#585858"})
	if m.done {
		if m.err != nil {
			footerText := fmt.Sprintf("Error: %v", m.err)
			styledFooter := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3333"))
			if m.width > 0 {
				styledFooter = styledFooter.Width(m.width - 4)
			}
			s.WriteString(styledFooter.Render(footerText) + "\n")
		} else {
			s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF66")).Render("Plan execution completed.") + "\n")
		}
	} else {
		s.WriteString(footerStyle.Render("Press 'q' or 'Ctrl+C' to cancel") + "\n")
	}

	return s.String()
}

func runWithBubbleTea(ctx *util.FragsContext, sm frags.SessionManager, paramsMap map[string]any, toolConfig ExtendedToolsConfig, args []string) (util.ProgMap, error) {
	eventChan := make(chan log.Event, 200)
	resChan := make(chan runResult, 1)

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	channelLevel := log.InfoChannelLevel
	if debug {
		channelLevel = log.DebugChannelLevel
	}
	streamerLogger := log.NewStreamerLogger(discardLogger, eventChan, channelLevel)

	go func() {
		defer close(eventChan)
		result, err := execute(ctx, sm, paramsMap, toolConfig,
			resources.NewFileResourceLoader(filepath.Dir(args[0])), streamerLogger)
		resChan <- runResult{result: result, err: err}
	}()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	m := model{
		spinner:        s,
		eventChan:      eventChan,
		resChan:        resChan,
		activeSessions: make(map[string]sessionState),
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	mResult := finalModel.(model)
	return mResult.result, mResult.err
}
