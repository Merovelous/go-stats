package models

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go-test/src/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	psnet "github.com/shirou/gopsutil/v3/net"
)

type NetTickMsg struct {
	id        int
	bytesRecv uint64
	bytesSent uint64
	timestamp time.Time
}

type SpeedtestMsg struct {
	Id       int
	Download float64
}

type SpeedtestTriggerMsg int

type NetworkModel struct {
	Id           int
	Interface    string
	NetType      string // "Wired" or "WiFi"
	WifiBand     string // "2.4GHz", "5GHz" or ""
	Ipv6Disabled bool

	DownloadRate float64 // Bytes per second
	UploadRate   float64 // Bytes per second

	lastBytesRecv uint64
	lastBytesSent uint64
	lastCheck     time.Time

	Polling bool

	SpeedtestDownload float64
	SpeedtestTime     string
	IsSpeedtesting    bool
}

func NewNetworkModel() NetworkModel {
	m := NetworkModel{
		Id:        0,
		Interface: "Detecting...",
		NetType:   "Unknown",
	}
	m.detectInterfaceInfo()
	return m
}

func (m *NetworkModel) detectInterfaceInfo() {
	// 1. Find default interface using `ip route`
	// ip route get 1.1.1.1 | grep -oP 'dev \K\S+'
	out, err := exec.Command("sh", "-c", "ip route get 1.1.1.1 | grep -oP 'dev \\K\\S+'").Output()
	if err == nil {
		m.Interface = strings.TrimSpace(string(out))
	} else {
		m.Interface = "eth0" // Fallback
	}

	// 2. Check if wireless
	// Check if /sys/class/net/<iface>/wireless exists
	_, err = os.ReadDir("/sys/class/net/" + m.Interface + "/wireless")
	if err == nil {
		m.NetType = "WiFi"
		// Try to find frequency using iw
		// iw dev <iface> link
		iwOut, _ := exec.Command("iw", "dev", m.Interface, "link").Output()
		iwStr := string(iwOut)
		if strings.Contains(iwStr, "5.0 MHz") || strings.Contains(iwStr, "5180") || strings.Contains(iwStr, "freq: 5") {
			m.WifiBand = "5GHz"
		} else if strings.Contains(iwStr, "freq: 2") {
			m.WifiBand = "2.4GHz"
		} else {
			m.WifiBand = "" // Unknown band
		}
	} else {
		m.NetType = "Wired"
	}

	// 3. Check IPv6
	// cat /proc/sys/net/ipv6/conf/all/disable_ipv6
	ipv6Out, err := os.ReadFile("/proc/sys/net/ipv6/conf/all/disable_ipv6")
	if err == nil {
		if strings.TrimSpace(string(ipv6Out)) == "1" {
			m.Ipv6Disabled = true
		} else {
			m.Ipv6Disabled = false
		}
	}
}

func (m NetworkModel) Init() tea.Cmd {
	if m.Polling {
		// Initialize counters lightly to avoid massive spike on first tick
		counters, _ := psnet.IOCounters(true)
		for _, c := range counters {
			if c.Name == m.Interface {
				m.lastBytesRecv = c.BytesRecv
				m.lastBytesSent = c.BytesSent
			}
		}
		m.lastCheck = time.Now()
		m.IsSpeedtesting = true
		return tea.Batch(
			getNetworkTick(m.Id, m.Interface),
			runSpeedtest(m.Id),
		)
	}
	return nil
}

func (m NetworkModel) Update(msg tea.Msg) (NetworkModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SpeedtestMsg:
		if msg.Id != m.Id {
			return m, nil
		}
		m.SpeedtestDownload = msg.Download
		m.SpeedtestTime = time.Now().Format("15:04")
		m.IsSpeedtesting = false
		// Schedule next one in 5 minutes
		return m, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
			return SpeedtestTriggerMsg(m.Id)
		})
	case SpeedtestTriggerMsg:
		if int(msg) != m.Id {
			return m, nil
		}
		if !m.Polling {
			return m, nil
		}
		m.IsSpeedtesting = true
		return m, runSpeedtest(m.Id)
	case NetTickMsg:
		if msg.id != m.Id {
			return m, nil
		}

		// Calculate rates
		duration := msg.timestamp.Sub(m.lastCheck).Seconds()
		if duration > 0 {
			m.DownloadRate = float64(msg.bytesRecv-m.lastBytesRecv) / duration
			m.UploadRate = float64(msg.bytesSent-m.lastBytesSent) / duration
		}

		m.lastBytesRecv = msg.bytesRecv
		m.lastBytesSent = msg.bytesSent
		m.lastCheck = msg.timestamp

		if m.Polling {
			return m, getNetworkTick(m.Id, m.Interface)
		}
	}
	return m, nil
}

func (m NetworkModel) View() string {
	title := styles.TitleStyle.Render("NETWORK DETAILS")

	ipv6Status := styles.StatValueStyle.Foreground(styles.ColorSuccess).Render("Disabled")
	if !m.Ipv6Disabled {
		// Enabled is considered "bad" by user implying they want it checked?
		// "ipv6 should be red if enabled" -> Yes.
		ipv6Status = styles.StatValueStyle.Foreground(styles.ColorError).Render("Enabled")
	}

	netTypeDisplay := m.NetType
	if m.NetType == "WiFi" && m.WifiBand != "" {
		netTypeDisplay += fmt.Sprintf(" (%s)", m.WifiBand)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		styles.RenderStat("󰈀 Interface:", m.Interface),
		styles.RenderStat(" Type:", netTypeDisplay),
		lipgloss.JoinHorizontal(lipgloss.Left, styles.StatKeyStyle.Render("󰅐 IPv6:"), ipv6Status),
		"",
		styles.RenderStat(" Download:", formatSpeed(m.DownloadRate)),
		styles.RenderStat(" Upload:", formatSpeed(m.UploadRate)),
		"",
		styles.RenderStat(" Total Rx:", formatSize(m.lastBytesRecv)),
		styles.RenderStat(" Total Tx:", formatSize(m.lastBytesSent)),
		"",
		m.renderSpeedtestSection(),
	)

	box := styles.StatBoxStyle.Render(content)
	// help := styles.HelpStyle.Render("[Space] Return to Menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

func (m NetworkModel) renderSpeedtestSection() string {
	if m.IsSpeedtesting {
		return styles.StatKeyStyle.Render("󰾆 Speedtest: Running...")
	}
	if m.SpeedtestTime == "" {
		return styles.StatKeyStyle.Render("󰾆 Speedtest: Waiting...")
	}

	val := fmt.Sprintf("%.2f Mbps", m.SpeedtestDownload)
	timeStr := fmt.Sprintf("(at %s)", m.SpeedtestTime)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		styles.StatKeyStyle.Render("󰾆 Speedtest:"),
		styles.StatValueStyle.Foreground(styles.ColorCyan).Render(val),
		" ",
		styles.HelpStyle.Margin(0, 0).Render(timeStr),
	)
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.2f KB/s", bytesPerSec/1024)
	} else {
		return fmt.Sprintf("%.2f MB/s", bytesPerSec/(1024*1024))
	}
}

func formatSize(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	}
}

func getNetworkTick(id int, iface string) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return collectNetworkData(id, iface)
	})
}

func collectNetworkData(id int, iface string) tea.Msg {
	counters, _ := psnet.IOCounters(true)
	var recv, sent uint64
	for _, c := range counters {
		if c.Name == iface {
			recv = c.BytesRecv
			sent = c.BytesSent
			break
		}
	}
	return NetTickMsg{
		id:        id,
		bytesRecv: recv,
		bytesSent: sent,
		timestamp: time.Now(),
	}
}

func runSpeedtest(id int) tea.Cmd {
	return func() tea.Msg {
		// speedtest-cli --csv
		out, err := exec.Command("speedtest-cli", "--csv", "--no-upload", "--server", "17391").Output()
		if err != nil {
			return SpeedtestMsg{Id: id, Download: 0}
		}

		fields := strings.Split(string(out), ",")
		if len(fields) < 7 {
			return SpeedtestMsg{Id: id, Download: 0}
		}

		// Download speed is 7th field in bits/s
		downloadBits, err := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)
		if err != nil {
			return SpeedtestMsg{Id: id, Download: 0}
		}

		return SpeedtestMsg{
			Id:       id,
			Download: downloadBits / 1000000.0, // convert to Mbps
		}
	}
}
