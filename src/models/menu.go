package models

import (
	"fmt"
	"go-test/src/styles"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainModel struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected

	Page string // "menu" or "cpu" or "gpu"

	width  int
	height int

	cpuModel     CpuModel
	gpuModel     GpuModel
	netModel     NetworkModel
	procModel    ProcessModel
	spinnerIndex int
	currentTime  time.Time
}

type HeartbeatMsg time.Time

func InitialModel() MainModel {
	return MainModel{
		// Our to-do list is a grocery list
		choices: []string{"all", "network", "cpu", "gpu", "processes"},

		// A map which indicates which choices are selected. We're using
		// the map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
		Page:     "menu", // Start at the menu

		cpuModel:     NewCpuModel(),
		gpuModel:     NewGpuModel(),
		netModel:     NewNetworkModel(),
		procModel:    NewProcessModel(),
		spinnerIndex: 0,
	}
}

func doHeartbeat() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return HeartbeatMsg(t)
	})
}

func (m MainModel) Init() tea.Cmd {
	// Trigger a single initial data fetch for all models
	return tea.Batch(
		func() tea.Msg { return collectCpuData(m.cpuModel.Id) },
		func() tea.Msg { return collectNvidiaData(m.gpuModel.Id) },
		func() tea.Msg { return collectNetworkData(m.netModel.Id, m.netModel.Interface) },
		func() tea.Msg { return collectProcessData(m.procModel.Id) },
		doHeartbeat(),
	)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// Handle globally necessary messages like resize
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case HeartbeatMsg:
		m.spinnerIndex++
		m.currentTime = time.Time(msg)
		return m, doHeartbeat()
	}

	// --- CPU PAGE LOGIC ---
	if m.Page == "cpu" {
		// Handle return to menu
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
			m.Page = "menu"
			m.cpuModel.Polling = false
			return m, nil
		}

		var cmd tea.Cmd
		m.cpuModel, cmd = m.cpuModel.Update(msg)
		return m, cmd
	}

	// --- GPU PAGE LOGIC ---
	if m.Page == "gpu" {
		// Handle return to menu
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
			m.Page = "menu"
			m.gpuModel.Polling = false
			return m, nil
		}

		var cmd tea.Cmd
		m.gpuModel, cmd = m.gpuModel.Update(msg)
		return m, cmd
	}

	// --- NETWORK PAGE LOGIC ---
	if m.Page == "network" {
		// Handle return to menu
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
			m.Page = "menu"
			m.netModel.Polling = false
			return m, nil
		}

		var cmd tea.Cmd
		m.netModel, cmd = m.netModel.Update(msg)
		return m, cmd
	}

	// --- PROCESSES PAGE LOGIC ---
	if m.Page == "processes" {
		// Handle return to menu
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
			m.Page = "menu"
			m.procModel.Polling = false
			return m, nil
		}

		var cmd tea.Cmd
		m.procModel, cmd = m.procModel.Update(msg)
		return m, cmd
	}

	// --- ALL PAGE LOGIC ---
	if m.Page == "all" {
		// Handle return to menu
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
			m.Page = "menu"
			m.cpuModel.Polling = false
			m.gpuModel.Polling = false
			m.netModel.Polling = false
			m.procModel.Polling = false
			return m, nil
		}

		// Update all models
		var cmdC, cmdG, cmdN, cmdP tea.Cmd
		m.cpuModel, cmdC = m.cpuModel.Update(msg)
		m.gpuModel, cmdG = m.gpuModel.Update(msg)
		m.netModel, cmdN = m.netModel.Update(msg)
		m.procModel, cmdP = m.procModel.Update(msg)

		return m, tea.Batch(cmdC, cmdG, cmdN, cmdP)
	}

	// --- MENU PAGE LOGIC ---
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			m.Page = m.choices[m.cursor]
			// Start polling logic
			if m.Page == "cpu" {
				m.cpuModel.Polling = true
				m.cpuModel.Id++
				return m, m.cpuModel.Init()
			}
			if m.Page == "gpu" {
				m.gpuModel.Polling = true
				m.gpuModel.Id++
				return m, m.gpuModel.Init()
			}
			if m.Page == "network" {
				m.netModel.Polling = true
				m.netModel.Id++
				return m, m.netModel.Init()
			}
			if m.Page == "processes" {
				m.procModel.Polling = true
				m.procModel.Id++
				return m, m.procModel.Init()
			}
			if m.Page == "all" {
				m.cpuModel.Polling = true
				m.cpuModel.Id++

				m.gpuModel.Polling = true
				m.gpuModel.Id++

				m.netModel.Polling = true
				m.netModel.Id++

				m.procModel.Polling = true
				m.procModel.Id++

				return m, tea.Batch(
					m.cpuModel.Init(),
					m.gpuModel.Init(),
					m.netModel.Init(),
					m.procModel.Init(),
				)
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case " ":
			// Toggle selection (single select)
			for i := range m.selected {
				if i == m.cursor {
					continue
				}
				delete(m.selected, i)
			}
			m.selected[m.cursor] = struct{}{}
		}
	}
	return m, nil
}

func (m MainModel) View() string {

	var content string

	switch m.Page {
	case "cpu":
		content = m.cpuModel.View()
	case "gpu":
		content = m.gpuModel.View()
	case "network":
		content = m.netModel.View()
	case "processes":
		content = m.procModel.View()
	case "all":
		// Compose 2x2 grid
		row1 := lipgloss.JoinHorizontal(lipgloss.Top, m.cpuModel.View(), m.gpuModel.View())
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, m.netModel.View(), m.procModel.View())
		content = lipgloss.JoinVertical(lipgloss.Left, row1, row2)
	default:
		s := styles.MenuTitleStyle.Render("What data would you like to see?") + "\n\n"
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			row := fmt.Sprintf("%s %s", cursor, choice)
			if m.cursor == i {
				s += styles.MenuItemSelectedStyle.Render(row) + "\n"
			} else {
				s += styles.MenuItemStyle.Render(choice) + "\n"
			}
		}
		s += styles.HelpStyle.Render("\n[Enter] Select • [q] Quit")
		content = s
	}

	// Add Heartbeat
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinner := spinnerFrames[m.spinnerIndex%len(spinnerFrames)]

	dateStr := m.currentTime.Format("2006-01-02 03:04:05 PM")
	pulseRender := styles.StatValueStyle.Foreground(styles.ColorSuccess).Render(" " + spinner + " " + dateStr)

	content = lipgloss.JoinVertical(lipgloss.Left, content, pulseRender)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			styles.DocStyle.Render(content),
		)
	}

	return styles.DocStyle.Render(content)
}
