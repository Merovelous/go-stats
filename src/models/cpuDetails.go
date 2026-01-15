package models

import (
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"go-test/src/styles"

	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
	cpu "github.com/shirou/gopsutil/v3/cpu"
	host "github.com/shirou/gopsutil/v3/host"
	mem "github.com/shirou/gopsutil/v3/mem"
)

type CpuStatsMsg struct {
	id          int
	cpuName     string
	cpuFreq     float64
	cpuUsage    float64
	cpuTemp     float64
	cpuFanSpeed string

	ramTotal       string
	ramUsed        string
	ramFree        string
	ramUsedPercent string
	ramFreePercent string
}

type CpuModel struct {
	Id          int
	CpuName     string
	CpuFreq     float64
	CpuUsage    float64
	CpuTemp     float64
	CpuFanSpeed string

	RamTotal       string
	RamUsed        string
	RamFree        string
	RamUsedPercent string
	RamFreePercent string

	Polling bool
}

func NewCpuModel() CpuModel {
	return CpuModel{
		Id:             0,
		CpuName:        "Loading...",
		CpuFanSpeed:    "Loading...",
		RamTotal:       "Loading...",
		RamUsed:        "Loading...",
		RamFree:        "Loading...",
		RamUsedPercent: "0%",
		RamFreePercent: "0%",
	}
}

func (m CpuModel) Init() tea.Cmd {
	if m.Polling {
		return getCpuStats(m.Id)
	}
	return nil
}

func (m CpuModel) Update(msg tea.Msg) (CpuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case CpuStatsMsg:
		if msg.id != m.Id {
			return m, nil
		}
		m.CpuName = msg.cpuName
		m.CpuFreq = msg.cpuFreq
		m.CpuUsage = msg.cpuUsage
		m.CpuTemp = msg.cpuTemp
		m.CpuFanSpeed = msg.cpuFanSpeed
		m.RamTotal = msg.ramTotal
		m.RamUsed = msg.ramUsed
		m.RamFree = msg.ramFree
		m.RamUsedPercent = msg.ramUsedPercent
		m.RamFreePercent = msg.ramFreePercent
		if m.Polling {
			return m, getCpuStats(m.Id)
		}
	}
	return m, nil
}

func (m CpuModel) View() string {
	title := styles.TitleStyle.Render("CPU DETAILS")

	// Calculate usage color
	usageColor := styles.ColorSuccess
	if m.CpuUsage > 50 {
		usageColor = styles.ColorWarning
	}
	if m.CpuUsage > 80 {
		usageColor = styles.ColorError
	}
	progress := styles.RenderProgressBar(20, m.CpuUsage)
	usageStr := styles.StatValueStyle.Foreground(usageColor).Render(fmt.Sprintf("%s %.2f%%", progress, m.CpuUsage))

	tempColor := styles.GetTempColor(m.CpuTemp)
	tempIcon := styles.GetTempIcon(m.CpuTemp)
	tempStr := styles.StatValueStyle.Foreground(tempColor).Render(fmt.Sprintf("%s %.2f °C", tempIcon, m.CpuTemp))

	modelInfo := fmt.Sprintf("%s @ %.2f MHz", m.CpuName, m.CpuFreq)

	content := lipgloss.JoinVertical(lipgloss.Left,
		styles.RenderStat(" Model:", modelInfo),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, styles.StatKeyStyle.Render("󰾆 Usage:"), usageStr),
		lipgloss.JoinHorizontal(lipgloss.Left, styles.StatKeyStyle.Render(" Temp:"), tempStr),
		styles.RenderStat("󰜮 Fan Speed:", m.CpuFanSpeed),
		"",
		styles.RenderStat(" RAM Total:", m.RamTotal+" MiB"),
		styles.RenderStat(fmt.Sprintf(" RAM Used (%s):", m.RamUsedPercent), m.RamUsed+" MiB"),
		styles.RenderStat(fmt.Sprintf(" RAM Free (%s):", m.RamFreePercent), m.RamFree+" MiB"),
	)

	box := styles.StatBoxStyle.Render(content)
	// help := styles.HelpStyle.Render("[Space] Return to Menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

func collectCpuData(id int) tea.Msg {
	var cpuName string
	var cpuFreq float64
	var cpuUsage float64
	var cpuTemp float64

	var cpuFanSpeed string = "N/A"

	var ramTotal string
	var ramUsed string
	var ramFree string

	// ... (cpu/mem logic is unchanged, I am targeting surrounding lines to insert)

	// Collect Fan Speed via sensors
	out, errSensors := exec.Command("sensors").Output()
	if errSensors == nil {
		re := regexp.MustCompile(`cpu_fan:\s+(\d+\s+RPM)`)
		matches := re.FindStringSubmatch(string(out))
		if len(matches) > 1 {
			cpuFanSpeed = matches[1]
		}
	}

	percent, err := cpu.Percent(0, false)
	if err != nil {
		cpuUsage = 0
	} else {
		cpuUsage = percent[0]
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		cpuName = "N/A"
		cpuFreq = 0
	} else {
		if len(cpuInfo) > 0 {
			cpuName = cpuInfo[0].ModelName
			cpuFreq = cpuInfo[0].Mhz
		} else {
			cpuName = "Unknown"
			cpuFreq = 0
		}
	}

	tempdata, err := host.SensorsTemperatures()
	if err != nil || len(tempdata) == 0 {
		cpuTemp = 0
	} else {
		cpuTemp = tempdata[0].Temperature
	}

	var ramUsedPercent string
	var ramFreePercent string

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		ramTotal = "N/A"
		ramUsed = "N/A"
		ramFree = "N/A"
		ramUsedPercent = "0%"
		ramFreePercent = "0%"
	} else {
		ramTotal = fmt.Sprintf("%.2f", float64(vmStat.Total)/1024/1024) // in MiB
		ramUsed = fmt.Sprintf("%.2f", float64(vmStat.Used)/1024/1024)
		ramFree = fmt.Sprintf("%.2f", float64(vmStat.Free)/1024/1024)

		usedP := float64(vmStat.Used) / float64(vmStat.Total) * 100
		freeP := float64(vmStat.Free) / float64(vmStat.Total) * 100
		ramUsedPercent = fmt.Sprintf("%.0f%%", usedP)
		ramFreePercent = fmt.Sprintf("%.0f%%", freeP)
	}

	return CpuStatsMsg{
		id:          id,
		cpuName:     cpuName,
		cpuFreq:     cpuFreq,
		cpuUsage:    cpuUsage,
		cpuTemp:     cpuTemp,
		cpuFanSpeed: cpuFanSpeed,

		ramTotal:       ramTotal,
		ramUsed:        ramUsed,
		ramFree:        ramFree,
		ramUsedPercent: ramUsedPercent,
		ramFreePercent: ramFreePercent,
	}
}

func getCpuStats(id int) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return collectCpuData(id)
	})
}
