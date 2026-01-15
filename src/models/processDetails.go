package models

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-test/src/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProcessItem struct {
	Pid   string
	Name  string
	Value float64
	Unit  string
}

type ProcessMsg struct {
	id     int
	CpuTop []ProcessItem
	RamTop []ProcessItem
}

type ProcessModel struct {
	Id      int
	CpuTop  []ProcessItem
	RamTop  []ProcessItem
	Polling bool
}

func NewProcessModel() ProcessModel {
	return ProcessModel{
		Id: 0,
	}
}

func (m ProcessModel) Init() tea.Cmd {
	if m.Polling {
		return getProcessStats(m.Id)
	}
	return nil
}

func (m ProcessModel) Update(msg tea.Msg) (ProcessModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ProcessMsg:
		if msg.id != m.Id {
			return m, nil
		}
		m.CpuTop = msg.CpuTop
		m.RamTop = msg.RamTop

		if m.Polling {
			return m, getProcessStats(m.Id)
		}
	}
	return m, nil
}

func (m ProcessModel) View() string {
	title := styles.TitleStyle.Render("TOP PROCESSES")

	// Render tables
	// Render tables
	t1 := renderTable(" Top CPU", "Usage %", m.CpuTop)
	t2 := renderTable(" Top RAM", "Usage %", m.RamTop)

	// Add separator
	t1 = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(styles.ColorSubtext).
		PaddingRight(1).
		Render(t1)

	t2 = lipgloss.NewStyle().PaddingLeft(1).Render(t2)

	// Grid layout
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, t1, t2)
	grid := lipgloss.JoinVertical(lipgloss.Left, topRow)

	box := styles.StatBoxStyle.Render(grid)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

func renderTable(titleStr, valueLabel string, items []ProcessItem) string {
	// Column widths
	nameWidth := 20
	pidWidth := 10
	valWidth := 10

	// Column styles
	nameCol := styles.TableCellStyle.Width(nameWidth)
	pidCol := styles.TableCellStyle.Width(pidWidth)
	valCol := styles.TableCellStyle.Width(valWidth).Align(lipgloss.Right)

	headerStyle := styles.TableHeaderStyle.Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(styles.ColorSubtext)

	// Header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(nameWidth).Render("Name"),
		headerStyle.Width(pidWidth).Render("PID"),
		headerStyle.Width(valWidth).Align(lipgloss.Right).Render(valueLabel),
	)

	// Title + Header + Rows
	rows := []string{
		styles.StatKeyStyle.Bold(true).Foreground(styles.ColorPrimary).MarginBottom(1).Render(titleStr),
		header,
	}

	if len(items) == 0 {
		rows = append(rows, styles.StatKeyStyle.Render("No data..."))
		rows = append(rows, styles.StatKeyStyle.Render("No data..."))
		rows = append(rows, styles.StatKeyStyle.Render("No data..."))
		rows = append(rows, styles.StatKeyStyle.Render("No data..."))
		rows = append(rows, styles.StatKeyStyle.Render("No data..."))
	}

	for i, item := range items {
		if i >= 5 {
			break
		}
		name := item.Name
		if len(name) > nameWidth-2 {
			name = name[:nameWidth-3] + "…"
		}

		valStr := fmt.Sprintf("%.1f", item.Value)

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			nameCol.Render(name),
			pidCol.Render(item.Pid),
			valCol.Render(valStr),
		)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func collectProcessData(id int) tea.Msg {
	cpuTop := fetchPsData("pcpu")
	ramTop := fetchPsData("pmem")

	return ProcessMsg{
		id:     id,
		CpuTop: cpuTop,
		RamTop: ramTop,
	}
}

func fetchPsData(sortKey string) []ProcessItem {
	cmd := exec.Command("ps", "-eo", "pid,comm,"+sortKey, "--no-headers")
	out, err := cmd.Output()
	if err != nil {
		return []ProcessItem{}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var items []ProcessItem
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			val, _ := strconv.ParseFloat(fields[2], 64)
			items = append(items, ProcessItem{
				Pid:   fields[0],
				Name:  fields[1],
				Value: val,
				Unit:  "%",
			})
		}
	}

	// Sort descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})

	// Take top 5
	if len(items) > 5 {
		return items[:5]
	}
	return items
}

func getProcessStats(id int) tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return collectProcessData(id)
	})
}
