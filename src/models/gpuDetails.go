package models

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	styles "go-test/src/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GpuStatsMsg struct {
	id                   int
	gpuName              string
	gpuUsage             string
	gpuTemp              string
	gpuFans              string
	gpuMemoryTotal       string
	gpuMemoryUsed        string
	gpuMemoryFree        string
	gpuMemoryUsedPercent string
	gpuMemoryFreePercent string
}

type GpuModel struct {
	Id                   int
	GpuName              string
	GpuUsage             string
	GpuTemp              string
	GpuFans              string
	GpuMemoryTotal       string
	GpuMemoryUsed        string
	GpuMemoryFree        string
	GpuMemoryUsedPercent string
	GpuMemoryFreePercent string

	Polling bool
}

func NewGpuModel() GpuModel {
	return GpuModel{
		Id:                   0,
		GpuName:              "Loading...",
		GpuUsage:             "Loading...",
		GpuTemp:              "Loading...",
		GpuFans:              "Loading...",
		GpuMemoryTotal:       "Loading...",
		GpuMemoryUsed:        "Loading...",
		GpuMemoryFree:        "Loading...",
		GpuMemoryUsedPercent: "0%",
		GpuMemoryFreePercent: "0%",
	}
}

func (m GpuModel) Init() tea.Cmd {
	if m.Polling {
		return getGpuStats(m.Id)
	}
	return nil
}

func (m GpuModel) Update(msg tea.Msg) (GpuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case GpuStatsMsg:
		if msg.id != m.Id {
			return m, nil
		}
		m.GpuName = msg.gpuName
		m.GpuUsage = msg.gpuUsage
		m.GpuTemp = msg.gpuTemp
		m.GpuFans = msg.gpuFans
		m.GpuMemoryTotal = msg.gpuMemoryTotal
		m.GpuMemoryUsed = msg.gpuMemoryUsed
		m.GpuMemoryFree = msg.gpuMemoryFree
		m.GpuMemoryUsedPercent = msg.gpuMemoryUsedPercent
		m.GpuMemoryFreePercent = msg.gpuMemoryFreePercent
		if m.Polling {
			return m, getGpuStats(m.Id)
		}
	}
	return m, nil
}

func (m GpuModel) View() string {
	title := styles.TitleStyle.Render("GPU DETAILS")

	// Parse temp for color
	tempVal, _ := strconv.ParseFloat(m.GpuTemp, 64)
	tempColor := styles.GetTempColor(tempVal)
	tempIcon := styles.GetTempIcon(tempVal)
	tempStr := styles.StatValueStyle.Foreground(tempColor).Render(fmt.Sprintf("%s %s°C", tempIcon, m.GpuTemp))

	usageVal, _ := strconv.ParseFloat(m.GpuUsage, 64)
	progress := styles.RenderProgressBar(20, usageVal)
	usageStr := fmt.Sprintf("%s %s%%", progress, m.GpuUsage)

	fanStr := m.GpuFans
	if !strings.Contains(fanStr, "RPM") && !strings.Contains(fanStr, "N/A") {
		fanStr += "%"
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		styles.RenderStat("󰍛 Model:", m.GpuName),
		"",
		styles.RenderStat("󰾆 Usage:", usageStr),
		lipgloss.JoinHorizontal(lipgloss.Left, styles.StatKeyStyle.Render(" Temp:"), tempStr),
		styles.RenderStat("󰜮 Fans:", fanStr),
		"",
		styles.RenderStat(" Memory Total:", m.GpuMemoryTotal+" MiB"),
		styles.RenderStat(fmt.Sprintf(" Memory Used (%s):", m.GpuMemoryUsedPercent), m.GpuMemoryUsed+" MiB"),
		styles.RenderStat(fmt.Sprintf(" Memory Free (%s):", m.GpuMemoryFreePercent), m.GpuMemoryFree+" MiB"),
	)

	box := styles.StatBoxStyle.Render(content)
	// help := styles.HelpStyle.Render("[Space] Return to Menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

func collectNvidiaData(id int) tea.Msg {
	// Optimization: Fetch all data in one command
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,utilization.gpu,temperature.gpu,fan.speed,memory.total,memory.used,memory.free", "--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return GpuStatsMsg{
			id:                   id,
			gpuName:              "N/A",
			gpuUsage:             "N/A",
			gpuTemp:              "N/A",
			gpuFans:              "N/A",
			gpuMemoryTotal:       "N/A",
			gpuMemoryUsed:        "N/A",
			gpuMemoryFree:        "N/A",
			gpuMemoryUsedPercent: "0%",
			gpuMemoryFreePercent: "0%",
		}
	}

	csv := strings.TrimSpace(string(out))
	fields := strings.Split(csv, ", ")

	if len(fields) < 7 {
		return GpuStatsMsg{
			id:                   id,
			gpuName:              "Error parsing",
			gpuUsage:             "N/A",
			gpuTemp:              "N/A",
			gpuFans:              "N/A",
			gpuMemoryTotal:       "N/A",
			gpuMemoryUsed:        "N/A",
			gpuMemoryFree:        "N/A",
			gpuMemoryUsedPercent: "0%",
			gpuMemoryFreePercent: "0%",
		}
	}

	gpuFans := fields[3]

	// Overwrite with sensors data if available
	outSensors, errSensors := exec.Command("sensors").Output()
	if errSensors == nil {
		re := regexp.MustCompile(`gpu_fan:\s+(\d+\s+RPM)`)
		matches := re.FindStringSubmatch(string(outSensors))
		if len(matches) > 1 {
			gpuFans = matches[1]
		}
	}

	// Calculate Percentages
	var memUsedPercent, memFreePercent string = "0%", "0%"
	totalVal, _ := strconv.ParseFloat(fields[4], 64)
	usedVal, _ := strconv.ParseFloat(fields[5], 64)
	freeVal, _ := strconv.ParseFloat(fields[6], 64)

	if totalVal > 0 {
		memUsedPercent = fmt.Sprintf("%.0f%%", (usedVal/totalVal)*100)
		memFreePercent = fmt.Sprintf("%.0f%%", (freeVal/totalVal)*100)
	}

	return GpuStatsMsg{
		id:                   id,
		gpuName:              fields[0],
		gpuUsage:             fields[1],
		gpuTemp:              fields[2],
		gpuFans:              gpuFans,
		gpuMemoryTotal:       fields[4],
		gpuMemoryUsed:        fields[5],
		gpuMemoryFree:        fields[6],
		gpuMemoryUsedPercent: memUsedPercent,
		gpuMemoryFreePercent: memFreePercent,
	}
}

func getGpuStats(id int) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return collectNvidiaData(id)
	})
}
